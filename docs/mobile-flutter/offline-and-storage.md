# Локальное хранилище Hive, Cache-First и Сетевой слой

Мобильное приложение **Orbit Mobile** спроектировано по принципу **Offline-First**: даже при полном отсутствии интернета или временной недоступности бэкенда студент может мгновенно открыть расписание своей группы и свои локальные записи.

---

## 1. Единое локальное хранилище: Hive

В качестве единого стандарта локального NoSQL-хранилища в проекте принята библиотека **`hive_ce`** (`hive_ce_flutter`) — поддерживаемый сообществом форк Hive с полной поддержкой Dart 3.5+:

* **Почему Hive:** Высокая скорость чтения бинарных данных при холодном старте, отсутствие нативных зависимостей C++ SQLite и детерминированная сериализация.

### Структура коробок (Boxes):
1. **`schedule_cache_box`**: Временный кэш сеток расписания (восстанавливаемый).
   * **Формат ключа:** `schedule:{group}:week:{weekStartIso}` (пример: `schedule:261-334:week:2026-10-26`).  
   * **Архитектурное примечание:** API Gateway поддерживает запросы произвольных диапазонов дат до 93 дней через параметры `?date_from=YYYY-MM-DD&date_to=YYYY-MM-DD`. Нарезка кэша по неделям — это осознанная стратегия мобильного клиента для быстрого пагинирования в горизонтальном таймлайне и экономии оперативной памяти мобильного устройства.
2. **`app_meta_box`**: Нетипизированная коробка служебных метаданных (`schema_version_<boxName>`, временные метки кэша).
3. **`settings_box`**: Гостевые и пользовательские настройки (`selected_group`, `is_onboarding_completed`, `theme_mode`).
4. **`user_tasks_box`** и **`user_notes_box`**: Личные пользовательские задачи и конспекты.

### Разделение миграций и защиты данных от сбоев:
* **Служебные метаданные:** Версии схем хранятся в отдельной коробке `app_meta_box` под строковыми ключами вида `schema_version_<boxName>`. Это исключает падения типизации вида `as T` внутри типизированных коробок `Box<T>`.
* **Политика миграций и сбоев:**
  * Для кэша расписания (`schedule_cache_box`): при повреждении файла или несовпадении версий кэш можно очистить и перезапросить с сервера.
  * Для пользовательских данных (`user_tasks_box`, `user_notes_box`): **автоматическое удаление коробки категорически запрещено**. Если коробка пуста (первый запуск), версия схемы инициализируется автоматически. Если же в коробке есть данные старой версии, **мигратор строго обязателен**: при его отсутствии выбрасывается `StateError`, а версия в `app_meta_box` не меняется.

```dart
// lib/core/storage/hive_storage.dart
import 'package:hive_ce_flutter/hive_ce_flutter.dart';

typedef HiveMigrator<T> = Future<void> Function(Box<T> box, int fromVersion, int toVersion);

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

  /// Открытие пользовательских данных с обязательной проверкой мигратора
  static Future<Box<T>> openUserDataBox<T>(
    String boxName, {
    required int targetVersion,
    HiveMigrator<T>? migrator,
  }) async {
    final versionKey = 'schema_version_$boxName';
    final savedVersion = metaBox.get(versionKey, defaultValue: 0) as int;

    final box = await Hive.openBox<T>(boxName);

    // Сценарий 1: Первичное создание пустой коробки
    if (savedVersion == 0 && box.isEmpty) {
      await metaBox.put(versionKey, targetVersion);
      return box;
    }

    // Сценарий 2: Повышение версии схемы для существующих данных
    if (savedVersion < targetVersion) {
      if (migrator == null) {
        throw StateError(
          'Обнаружены существующие данные в $boxName (v$savedVersion), '
          'но мигратор на v$targetVersion не передан. Обновление версии отменено.',
        );
      }
      // Выполняем миграцию
      await migrator(box, savedVersion, targetVersion);
      // Фиксируем версию в мета-боксе ТОЛЬКО после успешной миграции
      await metaBox.put(versionKey, targetVersion);
    }

    return box;
  }
}
```

---

## 2. Стратегия Cache-First и согласование со статусами Gateway API

> **Правило сетевой изоляции:**  
> Мобильное приложение обращается **исключительно к `gateway-service`** (`GET /api/v1/schedule/groups/{group}`). Мобилка никогда не делает прямых запросов к сайту `rasp.dmami.ru` — нагрузку, защиту и опрос внешнего источника контролирует бэкенд `schedule-service`.

### Семантика серверного ответа Gateway:
Согласно контракту Gateway, все мета-поля проверки и синхронизации сгруппированы внутри объекта `metadata`:
```json
{
  "metadata": {
    "status": "READY",
    "is_stale": false,
    "last_checked_at": "2026-10-31T20:15:00Z"
  },
  "days": []
}
```

### Обработка статусов в мобильном клиенте:
1. **`status == "READY"`:**
   * Если массив `days` содержит занятия — готовое расписание сохраняется в Hive под недельным ключом.
   * Если `days` пустой (`days: []`) — **это допустимый штатный результат** (праздники, каникулы или экзаменационная сессия без пар). Интерфейс отображает состояние *«На этой неделе пар нет»*, а не сетевую ошибку!
2. **`status == "PENDING_SYNC"`:**
   * Сервер еще выполняет фоновый опрос источника.
   * **Критическое правило:** Ответ `PENDING_SYNC` **не должен заменять** уже имеющееся в Hive готовое расписание! Если кэш есть — продолжаем показывать его с плашкой *«Синхронизация на сервере...»*.
3. **Неизвестный статус (`status == "UNKNOWN"` или новый):**
   * Нельзя автоматически приравнивать к `READY`. Обрабатывается как временная ошибка обновления с сохранением локального кэша.
4. **Разграничение временных меток в приложении:**
   * `serverLastCheckedAt: DateTime?` — извлекается из `metadata.last_checked_at` (когда бэкенд проверял сайт вуза, может отсутствовать при статусе `PENDING_SYNC`).
   * `deviceUpdatedAt: DateTime` — момент локального сохранения/получения данных смартфоном (`DateTime.now()`).

---

## 3. Сетевой клиент Dio и канонический `AuthInterceptor`

Сетевое взаимодействие построено на библиотеке **`Dio 5.4+`**.

### Проектируемые правила интерцептора:
1. **Сравнение токена перед Refresh:** Если несколько параллельных запросов упали с 401, первый запрос запускает обновление. Когда второй запрос выходит из очереди `QueuedInterceptor`, он сначала сравнивает свой заголовок с актуальным токеном в `SecureStorage`. Если токен уже обновился — повторный запрос выполняется сразу без вызова refresh!
2. **Single-Flight Refresh (Мьютекс):** Единый `Future<String?>? _refreshFuture` гарантирует ровно один сетевой запрос `POST /api/v1/auth/refresh`.
3. **Устранение взаимной блокировки (Deadlock Prevention):**  
   Повторный запрос `fetch(retryOptions)` выполняется **через отдельный изолированный клиент `_tokenDio`**, на котором НЕТ `AuthInterceptor`. Это исключает повторное попадание упавшего повторного запроса в заблокированную очередь `QueuedInterceptor`.
4. **Раздельная обработка ошибок:** Если повторный запрос завершается ошибкой `403 Forbidden` (нет прав на ресурс), ошибка пробрасывается в UI через `handler.next(retryError)` и **не сбрасывает** токены сессии!
5. **Безопасность токенов при сбоях сети:** При сетевом таймауте (`ConnectionTimeout`, `SocketException`) токены **не удаляются**. Токены сбрасываются только при явном ответе сервера 401/403 на сам эндпоинт `/auth/refresh`.
6. **Сохранение гостевых настроек:** При выходе из аккаунта (Logout) очищаются только приватные токены в KeyStore/Keychain. Выбранная студентом группа и настройки в `settings_box` сохраняются.

```dart
// lib/core/network/interceptors/auth_interceptor.dart
import 'package:dio/dio.dart';
import '../../storage/secure_storage.dart';

class AuthInterceptor extends QueuedInterceptor {
  AuthInterceptor({
    required SecureStorage secureStorage,
    required Dio dio,
    required String tokenRefreshUrl,
  })  : _secureStorage = secureStorage,
        _dio = dio,
        _tokenRefreshUrl = tokenRefreshUrl,
        _tokenDio = Dio(BaseOptions(
          baseUrl: dio.options.baseUrl,
          connectTimeout: const Duration(seconds: 5),
          receiveTimeout: const Duration(seconds: 5),
        ));

  final SecureStorage _secureStorage;
  final Dio _dio;
  final String _tokenRefreshUrl;
  final Dio _tokenDio; // Изолированный клиент без AuthInterceptor

  Future<String?>? _refreshFuture;

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
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

    if (statusCode != 401 || isRetry || err.requestOptions.extra['requiresAuth'] == false) {
      return handler.next(err);
    }

    // 1. Проверяем: возможно, токен уже обновился параллельным запросом пока этот ждал в очереди
    final requestHeader = err.requestOptions.headers['Authorization'] as String?;
    final currentToken = await _secureStorage.getAccessToken();

    String? tokenToUse = currentToken;

    if (currentToken == null || requestHeader == 'Bearer $currentToken') {
      try {
        tokenToUse = await _performSingleFlightRefresh();
      } on DioException catch (refreshDioError) {
        final refreshCode = refreshDioError.response?.statusCode;
        if (refreshCode == 401 || refreshCode == 403) {
          // Сбрасываем токены ТОЛЬКО если сервер подтвердил недействительность Refresh токена
          await _secureStorage.clearTokens();
        }
        return handler.next(err);
      } catch (_) {
        return handler.next(err);
      }
    }

    if (tokenToUse == null) {
      return handler.next(err);
    }

    // 2. Повторяем исходный запрос через _tokenDio (предотвращает Deadlock в QueuedInterceptor)
    try {
      final retryOptions = err.requestOptions;
      retryOptions.headers['Authorization'] = 'Bearer $tokenToUse';
      retryOptions.extra['isRetry'] = true;

      final response = await _tokenDio.fetch(retryOptions);
      return handler.resolve(response);
    } on DioException catch (retryError) {
      // Ошибка ресурса (например 403 Forbidden) НЕ должна стирать валидные токены!
      return handler.next(retryError);
    }
  }

  Future<String?> _performSingleFlightRefresh() async {
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
      _refreshFuture = null;
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
