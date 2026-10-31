# Локальное хранилище Hive, Cache-First и Сетевой слой

Мобильное приложение **Orbit Mobile** спроектировано по принципу **Offline-First**: даже при полном отсутствии интернета или временной недоступности бэкенда студент может мгновенно открыть расписание своей группы и свои локальные записи.

---

## 1. Единое локальное хранилище: Hive

В качестве единого стандарта локального NoSQL-хранилища в проекте принята библиотека **`hive_ce`** (`hive_ce_flutter`) — поддерживаемый сообществом форк Hive с полной поддержкой Dart 3.5+:

* **Почему Hive:** Высокая скорость чтения бинарных данных при холодном старте, отсутствие нативных зависимостей C++ SQLite и детерминированная сериализация.

### Структура коробок (Boxes):
1. **`schedule_cache_box`**: Временный кэш сеток расписания (восстанавливаемый).
   * **Формат ключа:** `schedule:{group}:week:{weekStartIso}` (пример: `schedule:261-334:week:2026-10-26`). Ключ строго привязан к неделе, так как Gateway возвращает недельный диапазон, а не весь семестр сразу.
2. **`app_meta_box`**: Служебные метаданные (версии схем данных, временные метки кэша).
3. **`settings_box`**: Гостевые и пользовательские настройки (`selected_group`, `is_onboarding_completed`, `theme_mode`).
4. **`user_tasks_box`** и **`user_notes_box`**: Личные пользовательские задачи и конспекты.

### Разделение миграций и защиты данных от сбоев:
* **Служебные метаданные:** Версии схем хранятся в отдельной коробке `app_meta_box` под строковыми ключами вида `schema_version_<boxName>`. Это исключает падения типизации вида `as T` внутри типизированных коробок `Box<T>`.
* **Политика при сбоях (Corruption Policy):**
  * Для кэша расписания (`schedule_cache_box`): при повреждении файла или несовпадении версий кэш можно очистить и перезапросить с сервера.
  * Для пользовательских данных (`user_tasks_box`, `user_notes_box`): **автоматическое удаление коробки категорически запрещено**. При ошибках чтения данные блокируются в режиме read-only с оповещением пользователя и отправкой отчета в лог.

```dart
// lib/core/storage/hive_storage.dart
import 'package:hive_ce_flutter/hive_ce_flutter.dart';

class HiveStorage {
  static const int scheduleCacheSchemaVersion = 1;
  static const int userTasksSchemaVersion = 1;

  static late Box<dynamic> metaBox;

  static Future<void> init() async {
    await Hive.initFlutter();
    metaBox = await Hive.openBox<dynamic>('app_meta_box');
  }

  /// Открытие восстанавливаемого кэша расписания
  static Future<Box<T>> openCacheBox<T>(String boxName, {required int version}) async {
    final versionKey = 'schema_version_$boxName';
    final savedVersion = metaBox.get(versionKey, defaultValue: 0) as int;

    try {
      final box = await Hive.openBox<T>(boxName);
      if (savedVersion != version) {
        await box.clear();
        await metaBox.put(versionKey, version);
      }
      return box;
    } catch (_) {
      // Кэш расписания можно безопасно пересоздать с диска
      await Hive.deleteBoxFromDisk(boxName);
      await metaBox.put(versionKey, version);
      return Hive.openBox<T>(boxName);
    }
  }

  /// Открытие пользовательских данных (строгий режим: удаление запрещено)
  static Future<Box<T>> openUserDataBox<T>(String boxName, {required int version}) async {
    final versionKey = 'schema_version_$boxName';
    final savedVersion = metaBox.get(versionKey, defaultValue: 0) as int;

    // Пользовательские коробки не удаляются при сбоях вслепую
    final box = await Hive.openBox<T>(boxName);
    if (savedVersion != version) {
      // Здесь вызываются явные миграторы данных, а не box.clear()
      await metaBox.put(versionKey, version);
    }
    return box;
  }
}
```

---

## 2. Стратегия Cache-First и согласование с Gateway API

> **Правило сетевой изоляции:**  
> Мобильное приложение обращается **исключительно к `gateway-service`** (`GET /api/v1/schedule/groups/{group}`). Мобилка никогда не делает прямых запросов к сайту `rasp.dmami.ru` — нагрузку, защиту и опрос внешнего источника контролирует бэкенд `schedule-service`.

### Согласование с серверным контрактом:
Серверный ответ Gateway содержит важные контекстные поля:
* `status`: `READY` (данные готовы), `PENDING_SYNC` (сервер в процессе синхронизации);
* `is_stale: bool`: признак, что бэкенд вернул сохраненный кэш, пока опрашивает вуз;
* `server_last_checked_at`: точное время, когда сервер последний раз проверял сайт вуза.

В мобильном клиенте обязательно разделяются:
1. `deviceCachedAt`: время, когда телефон локально сохранил данные в Hive;
2. `serverLastCheckedAt`: серверная отметка актуальности расписания.

### Жизненный цикл обновления экрана:
1. При открытии экрана расписания **сразу** отображаются данные из Hive под ключом текущей недели `schedule:{group}:week:{weekStartIso}`.
2. В фоне инициируется сетевой запрос к Gateway без блокировки UI.
3. При успешном ответе (200 OK):
   * Ответ сериализуется в Hive.
   * Экран обновляется **без сброса выбранного дня недели и позиции скролла**.
   * Обновляется плашка времени: *«Проверено сервером: сегодня в 14:20»*.
4. При ошибке сети / 503 / оффлайне:
   * Сохраненные пары остаются на экране.
   * Показывается отметка: *«Офлайн-режим. Данные от 12:00»*.
   * Приложение не заменяет расписание экраном ошибки, если локальные данные есть.

---

## 3. Сетевой клиент Dio и канонический `AuthInterceptor`

Сетевое взаимодействие построено на библиотеке **`Dio 5.4+`**.

### Проектируемые правила интерцептора:
1. **Single-Flight Refresh (Мьютекс):** При одновременном получении HTTP 401 от нескольких параллельных запросов выполняется ровно **один** сетевой запрос на обновление токенов. Все параллельные запросы ожидают завершения единого `Future<String?>`.
2. **Защита от зацикливания (`isRetry`):** Исходный запрос с новым токеном повторяется не более **одного** раза с флагом `extra['isRetry'] = true`.
3. **Изоляция служебного Dio:** Запрос refresh выполняется через отдельный экземпляр `Dio` без интерцепторов авторизации.
4. **Разграничение типов ошибок:**
   * **Сетевая ошибка (SocketException, ConnectionTimeout):** Токены **НЕ** удаляются, сессия считается активной, запрос завершается сетевой ошибкой `NetworkFailure`.
   * **Ошибка сессии (Сервер вернул 401/403 на refresh):** Токены сбрасываются через `clearTokens()`, пользователь переводится в гостевой режим.
5. **Сохранение гостевых настроек:** При выходе из аккаунта (Logout) очищаются только приватные токены в KeyStore/Keychain. Выбранная студентом группа и локальные настройки в `settings_box` сохраняются.
6. **Защита логов от утечек:** Запрещено логировать токены, заголовки `Authorization` и тело личных сообщений ИИ.

```dart
// lib/core/network/interceptors/auth_interceptor.dart
import 'package:dio/dio.dart';
import '../../storage/secure_storage.dart';

class AuthInterceptor extends QueuedInterceptor {
  AuthInterceptor({
    required SecureStorage secureStorage,
    required String tokenRefreshUrl,
  })  : _secureStorage = secureStorage,
        _tokenRefreshUrl = tokenRefreshUrl,
        _tokenDio = Dio(BaseOptions(
          connectTimeout: const Duration(seconds: 5),
          receiveTimeout: const Duration(seconds: 5),
        ));

  final SecureStorage _secureStorage;
  final String _tokenRefreshUrl;
  final Dio _tokenDio;

  // Общий Future для объединения параллельных refresh-запросов (Single-Flight)
  Future<String?>? _refreshFuture;

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    // Публичные запросы расписания не требуют Bearer токена
    if (options.extra['requiresAuth'] == false) {
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
    final statusCode = err.response?.statusCode;
    final isRetry = err.requestOptions.extra['isRetry'] == true;

    // Перехватываем 401 только если запрос требует авторизации и еще не повторялся
    if (statusCode != 401 || isRetry || err.requestOptions.extra['requiresAuth'] == false) {
      return handler.next(err);
    }

    try {
      final newToken = await _performSingleFlightRefresh();
      if (newToken == null) {
        return handler.next(err);
      }

      // Повторяем исходный запрос ровно один раз
      final retryOptions = err.requestOptions;
      retryOptions.headers['Authorization'] = 'Bearer $newToken';
      retryOptions.extra['isRetry'] = true;

      final response = await _tokenDio.fetch(retryOptions);
      return handler.resolve(response);
    } on DioException catch (refreshDioError) {
      // Стираем токены ТОЛЬКО если сервер подтвердил недействительность сессии (401/403)
      final refreshCode = refreshDioError.response?.statusCode;
      if (refreshCode == 401 || refreshCode == 403) {
        await _secureStorage.clearTokens();
      }
      // При сетевом таймауте токены не трогаем — сессия валидна, просто нет сети
      return handler.next(err);
    } catch (_) {
      return handler.next(err);
    }
  }

  Future<String?> _performSingleFlightRefresh() async {
    // Если refresh уже запущен другим запросом — ждем его завершения
    if (_refreshFuture != null) {
      return _refreshFuture;
    }

    _refreshFuture = () async {
      final refreshToken = await _secureStorage.getRefreshToken();
      if (refreshToken == null) {
        await _secureStorage.clearTokens();
        return null;
      }

      final response = await _tokenDio.post<Map<String, dynamic>>(
        _tokenRefreshUrl,
        data: {'refresh_token': refreshToken},
      );

      final newAccessToken = response.data?['access_token'] as String?;
      final newRefreshToken = response.data?['refresh_token'] as String?;

      if (newAccessToken != null && newRefreshToken != null) {
        await _secureStorage.saveTokens(
          accessToken: newAccessToken,
          refreshToken: newRefreshToken,
        );
        return newAccessToken;
      }

      await _secureStorage.clearTokens();
      return null;
    }();

    try {
      return await _refreshFuture;
    } finally {
      _refreshFuture = null; // Освобождаем мьютекс
    }
  }
}
```

---

## 4. Способы авторизации и статус мобильного контракта

### 📌 Текущие поддерживаемые способы входа на бэкенде:
* **Email OTP**: Отправка одноразового 6-значного кода подтверждения на почту студента.
* **Google OAuth**: Вход через Google-аккаунт.

> **Важное разграничение:**  
> Вход через Google или Email подтверждает только личность пользователя, но **не подтверждает принадлежность к академической группе**. Учебную группу студент указывает самостоятельно при первичном онбординге или в настройках профиля.

### ⏳ Проектируемые возможности `[Проектируется]`:
* **Мобильный OAuth Redirect:** Возврат в приложение через Deep Link схему `orbit://auth/callback` с PKCE-валидацией на стороне мобильного клиента находится в статусе проектирования мобильного контракта.
* **Интеграция с Личным кабинетом вуза:** Автоматическая верификация группы и роли старосты через деканат/ЛК Политеха является будущей интеграцией и не входит в релизный контракт базовой авторизации.
