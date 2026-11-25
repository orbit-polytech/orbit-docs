# Локальное хранилище Hive, Cache-First и Сетевой слой

Приложение **Orbit** (`mobile`) спроектировано по принципу **Offline-First**: даже при отсутствии интернета или временной недоступности бэкенда студент может мгновенно открыть расписание своей группы и сохранённую историю диалогов с ассистентом.

---

## 1. Единое локальное хранилище: `HiveStorageService`

В проекте используется библиотека **`hive_ce` (`^2.22.0`)** / **`hive_ce_flutter` (`^2.5.0`)** — поддерживаемый форк Hive для Dart 3.

### Фактические коробки (Boxes) и ключи

Имена коробок и ключей централизованы в `lib/core/constants/storage_key.dart` (`StorageKeys`), а их открытие и очистка инкапсулированы в `HiveStorageService` (`lib/core/storage/hive_storage_service.dart`):

1. **`schedule_box`** (`StorageKeys.scheduleBox`): кэш недельных сеток расписания в JSON-формате (`ScheduleResponseModel`), сохранённый через `ScheduleLocalDataSource`.
2. **`settings_box`** (`StorageKeys.settingsBox`): локальные настройки и предпочтения (`selected_group_title`, `favorite_group_titles`, `is_welcome_completed`, а также параметры профиля/подсказок).
3. **`assistant_history_box`** (`StorageKeys.assistantHistoryBox`): локальный кэш диалогов с ИИ-ассистентом.

```dart
// lib/core/storage/hive_storage_service.dart
import 'package:hive_ce_flutter/hive_flutter.dart';

import '../constants/storage_key.dart';

class HiveStorageService {
  HiveStorageService({HiveInterface? hive}) : _hive = hive ?? Hive;

  final HiveInterface _hive;

  late Box<dynamic> scheduleBox;
  late Box<dynamic> settingsBox;
  late Box<dynamic> assistantHistoryBox;

  Future<void> init() async {
    try {
      await Hive.initFlutter();
    } catch (_) {}

    scheduleBox = await _hive.openBox<dynamic>(StorageKeys.scheduleBox);
    settingsBox = await _hive.openBox<dynamic>(StorageKeys.settingsBox);
    assistantHistoryBox =
        await _hive.openBox<dynamic>(StorageKeys.assistantHistoryBox);
  }

  Future<void> clearAll() async {
    await scheduleBox.clear();
    await settingsBox.clear();
    await assistantHistoryBox.clear();
  }

  Future<void> close() async {
    await scheduleBox.close();
    await settingsBox.close();
    await assistantHistoryBox.close();
  }
}
```

Для управления пользовательским кэшем из настроек дополнительно используется `HiveAppCacheService` (`lib/core/storage/app_cache_service.dart`), который подсчитывает количество закэшированных расписаний и при очистке кэша очищает `scheduleBox` и `assistantHistoryBox`, **не затрагивая** `settingsBox` (выбранную группу и избранное).

Токены доступа и обновления хранятся отдельно от Hive — в защищённом хранилище `FlutterSecureTokenStore` (`lib/features/auth/data/datasources/session/secure_token_store.dart`, поверх `flutter_secure_storage ^11.2.0` с `AndroidOptions()` и `IOSOptions(accessibility: KeychainAccessibility.first_unlock_this_device)`).

---

## 2. Стратегия Cache-First и согласование со статусами Gateway API

> **Правило сетевой изоляции:**  
> Приложение обращается **исключительно к `gateway-service`** (`GET /api/v1/schedule/groups/{group}`). Клиент никогда не делает прямых запросов к сайту `rasp.dmami.ru` — нагрузку и опрос внешнего источника контролирует бэкенд `schedule-service`.

### Семантика серверного ответа Gateway:
Согласно контракту Gateway, мета-поля синхронизации сгруппированы внутри объекта `metadata`:

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

### Обработка статусов в клиенте:
1. **`status == "READY"`:**
   * Если массив `days` содержит занятия — готовое расписание сохраняется в `schedule_box`.
   * Если `days` пустой (`days: []`) — это допустимый штатный результат (каникулы, праздники или сессия без пар), а не сетевая ошибка.
2. **`status == "PENDING_SYNC"`:**
   * Сервер ещё выполняет фоновый опрос источника.
   * Если в `schedule_box` уже есть готовый кэш — он остаётся на экране (`ScheduleLoaded`) с индикацией фонового обновления; если кэша нет, показывается `SchedulePendingSync`.
3. **Сетевая ошибка при наличии кэша:**
   * При сбое сети готовое расписание из `schedule_box` сохраняется на экране (`ScheduleLoaded` с `errorMessage`).

---

## 3. Сетевой клиент `DioClient` и `AuthInterceptor`

Сетевой слой построен на **`dio` (`^5.11.1`)** (`lib/core/network/dio_client.dart`). Для поддержки Web-сборки адаптер настраивается платформенно через условный экспорт `dio_adapter_config.dart` (`dio_adapter_stub.dart` / `dio_adapter_web.dart` с поддержкой `withCredentials` в браузере).

### Как устроен реальный `AuthInterceptor` (`lib/core/network/auth_interceptor.dart`)

`AuthInterceptor` наследуется от `Interceptor` и делегирует чтение/обновление токенов абстракции **`AuthTokenProvider`** (`lib/core/network/auth_token_provider.dart`):

1. **Пропуск авторизации (`_shouldSkipAuth`):** заголовок `Authorization` не добавляется, если в запросе передано `options.extra['skipAuth'] == true`, либо путь относится к публичным auth-маршрутам (`/api/v1/auth/email/send-code`, `/api/v1/auth/email/verify-code`, `/api/v1/auth/refresh`, `/api/v1/auth/logout`, `/api/v1/auth/oauth/...`).
2. **Подстановка токена (`onRequest`):** если токен есть и запрос не помечен флагом повторной попытки (`auth_interceptor_retried`), добавляется заголовок `Authorization: Bearer <accessToken>`.
3. **Обработка `401` (`onError`):**
   * Если код ответа не `401`, запрос уже повторялся (`auth_interceptor_retried`), включён `skipAuth` или сервер вернул признак обязательной повторной аутентификации (`isReauthRequiredResponse(err.response)` — код `REAUTH_REQUIRED` из `reauth_error_detector.dart`), refresh **не запускается**, ошибка сразу уходит дальше.
   * Перед вызовом `refreshTokenCredentials()` сравнивается неудавшийся заголовок с текущим токеном из `_tokenProvider.getCurrentTokenCredentials()`: если другой параллельный запрос уже успел обновить токен и он не истёк, используется свежий токен без лишнего сетевого refresh.
   * После успешного обновления запрос повторяется с флагом `extra['auth_interceptor_retried'] = true` через `_dio!.fetch(retryOptions)` (или инжектированный `retryRequest` в тестах).

```dart
// lib/core/network/auth_interceptor.dart (ключевая логика)
class AuthInterceptor extends Interceptor {
  AuthInterceptor({
    required this._tokenProvider,
    Dio? dio,
    Future<Response<dynamic>> Function(RequestOptions options)? retryRequest,
  }) : assert(
         dio != null || retryRequest != null,
         'Either dio or retryRequest must be provided',
       ),
       _dio = dio,
       _retryRequest = retryRequest;

  static const String _retryFlagKey = 'auth_interceptor_retried';

  final AuthTokenProvider _tokenProvider;
  final Dio? _dio;
  final Future<Response<dynamic>> Function(RequestOptions options)? _retryRequest;

  bool _shouldSkipAuth(RequestOptions options) {
    if (options.extra['skipAuth'] == true) {
      return true;
    }
    final normalizedPath = Uri.tryParse(options.path)?.path ?? options.path;
    return normalizedPath == ApiEndpoints.authSendCode ||
        normalizedPath == ApiEndpoints.authVerifyCode ||
        normalizedPath == ApiEndpoints.authRefresh ||
        normalizedPath == ApiEndpoints.authLogout ||
        normalizedPath.startsWith('/api/v1/auth/oauth/');
  }

  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    final statusCode = err.response?.statusCode;
    final options = err.requestOptions;

    if (statusCode != 401 ||
        _shouldSkipAuth(options) ||
        options.extra[_retryFlagKey] == true ||
        isReauthRequiredResponse(err.response)) {
      handler.next(err);
      return;
    }
    // ... сравнение текущего токена / вызов _tokenProvider.refreshTokenCredentials()
    // ... и повтор запроса с _retryFlagKey = true
  }
}
```

---

## 4. Способы авторизации и профиль

### Реализовано на бэкенде и в `mobile`
- **Email OTP:** отправка 6-значного кода на почту (`send-code` / `verify-code`) с таймером повторной отправки.
- **Google OAuth + PKCE:** открытие браузера (`oauth/google/login`), возврат в мобильное приложение по deep link **`ru.mospolytech.orbit://auth/callback`** (`app_links`) и обмен одноразового кода через `POST /api/v1/auth/oauth/exchange` (на Web также поддерживается cookie/callback flow).
- **Привязка/отвязка Google и смена почты:** в `SettingsScreen` доступны привязка и отвязка Google, `ChangeEmailSheet` и `ReauthSheet` для подтверждения свежей сессии (`REAUTH_REQUIRED`).
- **Синхронизация выбранных групп:** `primary_group` и `favorite_groups` синхронизируются с аккаунтом через `GET /api/v1/auth/me` и `PATCH /api/v1/auth/preferences` (`PreferencesSyncCoordinator`).
- **Профиль пользователя (`users-service`):** загрузка и редактирование профиля (`EditProfileSheet`, пресеты аватаров) через `UserProfileCubit`.

> **Важно:** вход по Email или Google подтверждает аккаунт, а учебную группу и курс студент выбирает сам при онбординге, в поиске групп или в редактировании профиля.

### В планах `[Запланировано]`
- **Интеграция с Личным кабинетом вуза:** автоматическое подтверждение учебной группы и роли старосты.
