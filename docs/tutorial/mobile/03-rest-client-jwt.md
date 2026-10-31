# 03. Подключение к Gateway через REST API с JWT

Мобильное приложение Orbit Mobile взаимодействует со всеми бэкенд-сервисами исключительно через **API Gateway** по протоколу HTTPS.

В этой главе мы реализуем сетевой слой на базе библиотеки **`Dio`**, защищённое хранение токенов через **`flutter_secure_storage`** и автоматическое обновление сессии (Refresh Token Flow) с помощью интерцепторов.

---

## 1. Аппаратное хранилище токенов (`FlutterSecureStorage`)

Критическая ошибка начинающих мобильных разработчиков — сохранение токенов авторизации в `SharedPreferences` (Android) или `NSUserDefaults` (iOS). В этих хранилищах данные лежат в открытом виде в песочнице приложения и уязвимы при наличии физического доступа к файловой системе устройства.

В Orbit Mobile используется **`flutter_secure_storage`**:
- На **Android** данные шифруются ключом в аппаратном модуле **Android KeyStore** (AES/GCM).
- На **iOS** данные защищаются системным **Keychain Services** (с флагом `kSecAttrAccessibleAfterFirstUnlock`).

```dart
// core/storage/token_storage.dart
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

abstract interface class TokenStorage {
  Future<String?> getAccessToken();
  Future<String?> getRefreshToken();
  Future<void> saveTokens({required String accessToken, required String refreshToken});
  Future<void> clear();
}

class SecureTokenStorageImpl implements TokenStorage {
  final FlutterSecureStorage _storage;

  static const _accessTokenKey = 'orbit_access_token';
  static const _refreshTokenKey = 'orbit_refresh_token';

  SecureTokenStorageImpl({FlutterSecureStorage? storage})
      : _storage = storage ??
            const FlutterSecureStorage(
              aOptions: AndroidOptions(encryptedSharedPreferences: true),
              iOptions: IOSOptions(accessibility: KeychainAccessibility.first_unlock),
            );

  @override
  Future<String?> getAccessToken() => _storage.read(key: _accessTokenKey);

  @override
  Future<String?> getRefreshToken() => _storage.read(key: _refreshTokenKey);

  @override
  Future<void> saveTokens({required String accessToken, required String refreshToken}) async {
    await _storage.write(key: _accessTokenKey, value: accessToken);
    await _storage.write(key: _refreshTokenKey, value: refreshToken);
  }

  @override
  Future<void> clear() async {
    await _storage.delete(key: _accessTokenKey);
    await _storage.delete(key: _refreshTokenKey);
  }
}
```

---

## 2. Настройка HTTP-клиента `Dio`

Для работы с сетью используется библиотека `Dio`.

> **Критическое правило безопасности логирования:**  
> В `LogInterceptor` флаги `requestBody` и `responseBody` должны быть установлены в `false`. Вывод тела запросов в консоль или внешние сборщики крашей приводит к утечке JWT-токенов, паролей и персональных сообщений пользователей.

```dart
// core/network/dio_client.dart
import 'package:dio/dio.dart';
import '../storage/token_storage.dart';
import 'auth_interceptor.dart';

Dio createDioClient({
  required String baseUrl,
  required TokenStorage tokenStorage,
}) {
  final dio = Dio(
    BaseOptions(
      baseUrl: baseUrl,
      connectTimeout: const Duration(seconds: 5),
      receiveTimeout: const Duration(seconds: 15),
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      },
    ),
  );

  dio.interceptors.addAll([
    AuthInterceptor(
      dio: dio,
      tokenStorage: tokenStorage,
      tokenRefreshUrl: '$baseUrl/api/v1/auth/refresh',
    ),
    LogInterceptor(
      requestHeader: false,
      responseHeader: false,
      requestBody: false, // Запрещено: защита от утечки паролей и токенов
      responseBody: false, // Запрещено: защита личных данных
    ),
  ]);

  return dio;
}
```

---

## 3. `AuthInterceptor`: Защита от двойного Refresh и повтора запроса

Интерцептор решает три ключевые проблемы сетевого слоя в мобильных приложениях:
1. **Проверка актуальности токена:** если несколько параллельных запросов упали с 401, первый запрос запускает refresh. Второй запрос, ожидавший в очереди `QueuedInterceptor`, перед запуском повторного refresh проверяет: не обновился ли уже токен в хранилище? Если обновился — он сразу повторяет запрос с новым токеном.
2. **Изоляция ошибок:** ошибка 403 (нет прав на ресурс) при повторе запроса передается в UI и **не сбрасывает** токены сессии.
3. **Устранение взаимной блокировки (Deadlock):** повторный запрос выполняется через изолированный `_tokenDio.fetch(retryOptions)` без интерцепторов. Это исключает повторное попадание в заблокированную очередь `QueuedInterceptor` при ошибке повтора.

```dart
// core/network/auth_interceptor.dart
import 'package:dio/dio.dart';
import '../storage/token_storage.dart';

class AuthInterceptor extends QueuedInterceptor {
  final Dio _dio;
  final TokenStorage _tokenStorage;
  final String _tokenRefreshUrl;
  final Dio _tokenDio;

  Future<String?>? _refreshFuture;

  AuthInterceptor({
    required Dio dio,
    required TokenStorage tokenStorage,
    required String tokenRefreshUrl,
  })  : _dio = dio,
        _tokenStorage = tokenStorage,
        _tokenRefreshUrl = tokenRefreshUrl,
        _tokenDio = Dio(BaseOptions(
          baseUrl: dio.options.baseUrl,
          connectTimeout: const Duration(seconds: 5),
          receiveTimeout: const Duration(seconds: 5),
        ));

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    // Не добавляем токен в открытые запросы и эндпоинты авторизации
    if (options.extra['requiresAuth'] == false ||
        options.path.contains('/auth/login') ||
        options.path.contains('/auth/refresh')) {
      return handler.next(options);
    }

    final token = await _tokenStorage.getAccessToken();
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }

  @override
  Future<void> onError(DioException err, ErrorInterceptorHandler handler) async {
    final statusCode = err.response?.statusCode;
    final isRetry = err.requestOptions.extra['isRetry'] == true;

    // Не перехватываем, если это не 401, либо запрос уже повторялся, либо запрос был публичным
    if (statusCode != 401 || isRetry || err.requestOptions.extra['requiresAuth'] == false) {
      return handler.next(err);
    }

    // 1. Проверяем: возможно, пока запрос ждал в очереди, токен уже обновился параллельным запросом
    final requestHeader = err.requestOptions.headers['Authorization'] as String?;
    final currentToken = await _tokenStorage.getAccessToken();

    String? tokenToUse = currentToken;

    if (currentToken == null || requestHeader == 'Bearer $currentToken') {
      try {
        tokenToUse = await _performSingleFlightRefresh();
      } on DioException catch (refreshErr) {
        final code = refreshErr.response?.statusCode;
        if (code == 401 || code == 403) {
          // Сбрасываем сессию только если сам Refresh токен недействителен
          await _tokenStorage.clear();
        }
        return handler.next(err);
      } catch (_) {
        return handler.next(err);
      }
    }

    if (tokenToUse == null) {
      return handler.next(err);
    }

    // 2. Повторяем исходный запрос через отдельный _tokenDio (предотвращает дедлок в QueuedInterceptor)
    try {
      final retryOptions = err.requestOptions;
      retryOptions.headers['Authorization'] = 'Bearer $tokenToUse';
      retryOptions.extra['isRetry'] = true;

      final cloneResponse = await _tokenDio.fetch(retryOptions);
      return handler.resolve(cloneResponse);
    } on DioException catch (retryError) {
      // Ошибка повторного запроса (например 403) НЕ должна очищать сессию!
      return handler.next(retryError);
    }
  }

  Future<String?> _performSingleFlightRefresh() async {
    if (_refreshFuture != null) {
      return _refreshFuture;
    }

    _refreshFuture = () async {
      final refreshToken = await _tokenStorage.getRefreshToken();
      if (refreshToken == null) {
        await _tokenStorage.clear();
        return null;
      }

      final response = await _tokenDio.post<Map<String, dynamic>>(
        _tokenRefreshUrl,
        data: {'refresh_token': refreshToken},
      );

      final newAccessToken = response.data?['access_token'] as String?;
      final newRefreshToken = response.data?['refresh_token'] as String?;

      if (newAccessToken != null && newRefreshToken != null) {
        await _tokenStorage.saveTokens(
          accessToken: newAccessToken,
          refreshToken: newRefreshToken,
        );
        return newAccessToken;
      }

      await _tokenStorage.clear();
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

## 4. Антипаттерны: как нельзя и как правильно

### ❌ Как нельзя:
```dart
// ОШИБКА 1: Включение логирования тела ответов в продакшене
LogInterceptor(requestBody: true, responseBody: true) // Утечка паролей и личных чатов!

// ОШИБКА 2: Повторный refresh без проверки, не обновился ли токен в очереди
await _performSingleFlightRefresh(); // Избыточные запросы при серии 401!

// ОШИБКА 3: Очистка токенов при любой ошибке (включая 403 на защищенном ресурсе)
catch (e) { await secureStorage.clear(); } // Студента выбивает из аккаунта при отказе в доступе!
```

### ✅ Как правильно:
```dart
// Токены надежно хранятся в KeyStore/Keychain с аппаратной поддержкой:
await tokenStorage.saveTokens(accessToken: acc, refreshToken: ref);

// Проверка токена перед повторным refresh предотвращает дублирование запросов:
if (currentToken == null || requestHeader == 'Bearer $currentToken') { ... }

// Изолированная обработка ошибок повтора:
catch (retryError) { return handler.next(retryError); }
```

---

## Резюме

1. Сетевое взаимодействие централизовано через клиент `Dio` и API Gateway.
2. Чувствительные токены сессии сохраняются в `FlutterSecureStorage`.
3. `AuthInterceptor` проверяет актуальность токена перед вызовом refresh и защищает от гонок и ложных сбросов сессии.
4. Логирование строго изолировано: тела запросов и авторизационные заголовки не выводятся в консоль.
