# Локальное хранилище Hive, Cache-First и Сетевой слой

Мобильное приложение **Orbit Mobile** спроектировано по принципу **Offline-First**: даже при полном отсутствии интернета или недоступности внешних серверов вуза студент может мгновенно открыть расписание своей группы и свои заметки.

---

## 1. Единое локальное хранилище: Hive

В качестве основного стандарта локального хранилища в проекте выбрана база данных **Hive** (`hive_ce` / `hive_flutter`):

* **Почему Hive:** Сверхбыстрое NoSQL key-value хранилище на чистом Dart. В отличие от тяжелых реляционных баз (SQLite), Hive сохраняет объекты в бинарном формате с минимальным временем холодного старта и не имеет проблем с компиляцией нативных C/C++ библиотек на разных версиях Android NDK / iOS SDK.

### Структура коробок (Boxes):
1. **`schedule_box`**: Сетки расписаний учебных групп.
   * **Формат ключа:** `schedule:{group}:{semester}:{type}` (например, `schedule:261-334:2026-1:regular`).
2. **`settings_box`**: Гостевые и пользовательские настройки.
   * Ключи: `selected_group`, `is_onboarding_completed`, `theme_mode`.
3. **`local_tasks_box`** и **`local_notes_box`**: Личные задачи и конспекты (Local-First).

### Версионирование схем и защита от сбоев (Corruption Recovery):
* В каждой коробке хранится мета-ключ `_schema_version: 1`.
* Если при обновлении приложения структура DTO изменилась, и десериализация падает, коробка расписания автоматически пересоздается без краша всего приложения:

```dart
// lib/core/storage/hive_storage.dart
import 'package:hive_flutter/hive_flutter.dart';

class HiveStorage {
  static const int currentSchemaVersion = 1;

  static Future<Box<T>> openSafeBox<T>(String boxName) async {
    try {
      final box = await Hive.openBox<T>(boxName);
      final version = box.get('_schema_version', defaultValue: currentSchemaVersion as T);
      if (version != currentSchemaVersion) {
        await box.clear();
        await box.put('_schema_version' as dynamic, currentSchemaVersion as dynamic);
      }
      return box;
    } catch (e) {
      // При повреждении файла базы — удаляем битый файл и создаем чистый
      await Hive.deleteBoxFromDisk(boxName);
      return Hive.openBox<T>(boxName);
    }
  }
}
```

---

## 2. Стратегия Cache-First для расписания

> **Критическое правило сетевого взаимодействия:**  
> Мобильное приложение обращается **исключительно к `gateway-service`** (`GET /api/v1/schedule/groups/{group}`). Мобилка никогда не делает прямых запросов к сайту `rasp.dmami.ru` — нагрузку и кэширование внешних источников контролирует бэкенд-сервис `schedule-service`.

### Алгоритм работы:
```text
[ Пользователь открывает экран расписания ]
                   │
                   ▼
       Проверка локального Hive
      ┌────────────┴────────────┐
   Есть кэш                 Кэш пуст
      │                         │
      ▼                         ▼
Мгновенный рендер пар     Показ скелетона
(ScheduleReady, 0 мс)   (ScheduleLoadingFirstTime)
      │                         │
      └────────────┬────────────┘
                   ▼
   Фоновый запрос к Gateway (Dio)
   GET /api/v1/schedule/groups/{group}
      ┌────────────┴────────────┐
  Успех (200 OK)          Ошибка сети / 503
      │                         │
      ▼                         ▼
• Обновление Hive        • Кэш остается на экране
• Бесшовный рендер       • Бейдж «Сохранено в 12:00»
• Скролл НЕ сбрасывается • refreshFailure в стейте
```

---

## 3. Сетевой клиент Dio и правила AuthInterceptor

Сетевой клиент инкапсулирован в `ApiClient` на базе `Dio 5.4+`.

### Железные правила `AuthInterceptor`:

1. **Публичное расписание без токена:** Запрос `GET /api/v1/schedule/groups/{group}` выполняется публично, заголовок `Authorization` в него не добавляется.
2. **Мьютекс на Refresh (Single-Flight):** Если одновременно несколько параллельных запросов завершились ошибкой `401 Unauthorized`, клиент блокирует очередь (`dio.interceptors.requestLock`) и запускает ровно **один** запрос обновления токена:
   ```dart
   POST /api/v1/auth/refresh
   ```
   Остальные запросы ждут новый Access Token и повторяются после успешного ответа.
3. **Лимит повтора запроса:** Исходный запрос с новым токеном повторяется **не более одного раза**.
4. **Защита от бесконечного цикла:** Если запрос обновления токена сам завершается с ошибкой (401/403/истек срок Refresh сессии), клиент прекращает попытки, вызывает `authBloc.add(AuthLoggedOut())` и переводит приложение в гостевой режим.
5. **Разделение данных при выходе (Logout):** При выходе очищаются только приватные токены в `FlutterSecureStorage` и личные облачные данные. Гостевые настройки (выбранная учебная группа) **сохраняются**.
6. **Защита логов от утечек:** `LoggingInterceptor` строго исключает вывод токенов и тел запросов с персональными данными:

```dart
// lib/core/network/interceptors/auth_interceptor.dart
import 'package:dio/dio.dart';
import '../../storage/secure_storage.dart';

class AuthInterceptor extends QueuedInterceptor {
  AuthInterceptor({
    required SecureStorage secureStorage,
    required Dio dio,
  })  : _secureStorage = secureStorage,
        _dio = dio;

  final SecureStorage _secureStorage;
  final Dio _dio;

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    // Публичные запросы расписания не требуют Bearer JWT
    if (!options.extra.containsKey('requiresAuth') || options.extra['requiresAuth'] == false) {
      return handler.next(options);
    }

    final token = await _secureStorage.getAccessToken();
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }

  @override
  Future<void> onError(DioException err, ErrorInterceptorHandler handler) async {
    if (err.response?.statusCode != 401) {
      return handler.next(err);
    }

    // Если 401 случился уже на самом refresh эндпоинте — сессия мертва, сбрасываем вход
    if (err.requestOptions.path.contains('/auth/refresh')) {
      await _secureStorage.clearTokens();
      return handler.next(err);
    }

    try {
      final refreshToken = await _secureStorage.getRefreshToken();
      if (refreshToken == null) {
        return handler.next(err);
      }

      // Выполняем Single-Flight запрос обновления токена через отдельный Dio экземпляр
      final refreshResponse = await _dio.post(
        '/api/v1/auth/refresh',
        data: {'refresh_token': refreshToken},
        options: Options(extra: {'requiresAuth': false}),
      );

      final newAccessToken = refreshResponse.data['access_token'] as String;
      final newRefreshToken = refreshResponse.data['refresh_token'] as String;

      await _secureStorage.saveTokens(
        accessToken: newAccessToken,
        refreshToken: newRefreshToken,
      );

      // Повторяем исходный запрос ровно 1 раз
      final retryOptions = err.requestOptions;
      retryOptions.headers['Authorization'] = 'Bearer $newAccessToken';

      final response = await _dio.fetch(retryOptions);
      return handler.resolve(response);
    } catch (refreshError) {
      await _secureStorage.clearTokens();
      return handler.next(err);
    }
  }
}
```

---

## 4. Мобильная авторизация через Google OAuth

Вход через Google в мобильном приложении использует протокол **OAuth 2.0 PKCE** с возвратом через Custom URL Scheme:

1. Мобильное приложение открывает системный браузер (через `url_launcher` / `flutter_custom_tabs`) по ссылке:
   `https://auth.orbit.polytech/oauth/google?redirect_uri=orbit://auth/callback&code_challenge=...`
2. После успешной авторизации Google перенаправляет на схему приложения `orbit://auth/callback?code=4/0A...`.
3. Мобильное приложение перехватывает Deep Link через `app_links` и обменивает `code` + `code_verifier` на пару JWT-токенов (`access_token` и `refresh_token`).
4. Токены сохраняются в аппаратном KeyStore (Android) / Keychain (iOS) через `FlutterSecureStorage`.
