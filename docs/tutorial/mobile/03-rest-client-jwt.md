# 03. Подключение к Gateway через REST API с JWT

Мобильное приложение Orbit AI взаимодействует со всеми бэкенд-сервисами исключительно через **API Gateway** по протоколу HTTPS.

В этой главе мы реализуем сетевой слой на базе библиотеки **`Dio`**, защищённое хранение токенов через **`flutter_secure_storage`** и автоматическое обновление сессии (Refresh Token Flow) с помощью интерцепторов.

---

## 1. Безопасное хранилище токенов (`FlutterSecureStorage`)

Критическая ошибка начинающих мобильных разработчиков — сохранение токенов авторизации в `SharedPreferences` (Android) или `NSUserDefaults` (iOS). В этих хранилищах данные лежат в открытом виде и легко извлекаются вредоносным ПО.

В Orbit AI используется **`flutter_secure_storage`**:
- На **Android** данные шифруются ключом в аппаратном модуле **Android KeyStore**.
- На **iOS** данные защищаются системным **Keychain Services**.

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

Для работы с сетью используется библиотека `Dio`, предоставляющая поддержку интерцепторов, пула соединений и типизированных ошибок:

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
    AuthInterceptor(dio: dio, tokenStorage: tokenStorage),
    LogInterceptor(
      requestHeader: false,
      responseHeader: false,
      requestBody: true,
      responseBody: true,
    ),
  ]);

  return dio;
}
```

---

## 3. `AuthInterceptor`: Автоматическая подстановка и Refresh токена

Интерцептор выполняет две ключевые функции:
1. Автоматически добавляет заголовок `Authorization: Bearer <token>` во все запросы.
2. При получении ошибки `401 Unauthorized` прозрачно запрашивает новый токен через `/api/v1/auth/refresh` и повторяет упавший запрос.

```dart
// core/network/auth_interceptor.dart
import 'package:dio/dio.dart';
import '../storage/token_storage.dart';

class AuthInterceptor extends QueuedInterceptor {
  final Dio _dio;
  final TokenStorage _tokenStorage;

  AuthInterceptor({required Dio dio, required TokenStorage tokenStorage})
      : _dio = dio,
        _tokenStorage = tokenStorage;

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    // Не добавляем токен в открытые эндпоинты авторизации
    if (!options.path.contains('/auth/login') && !options.path.contains('/auth/refresh')) {
      final token = await _tokenStorage.getAccessToken();
      if (token != null) {
        options.headers['Authorization'] = 'Bearer $token';
      }
    }
    handler.next(options);
  }

  @override
  Future<void> onError(DioException err, ErrorInterceptorHandler handler) async {
    // Если ошибка 401 произошла не на самом эндпоинте обновления токена:
    if (err.response?.statusCode == 401 && !err.requestOptions.path.contains('/auth/refresh')) {
      final refreshToken = await _tokenStorage.getRefreshToken();

      if (refreshToken != null) {
        try {
          // Создаем отдельный изолированный инстанс Dio без интерцепторов для запроса refresh
          final refreshDio = Dio(BaseOptions(baseUrl: _dio.options.baseUrl));
          final response = await refreshDio.post(
            '/api/v1/auth/refresh',
            data: {'refresh_token': refreshToken},
          );

          final newAccessToken = response.data['access_token'] as String;
          final newRefreshToken = response.data['refresh_token'] as String;

          await _tokenStorage.saveTokens(
            accessToken: newAccessToken,
            refreshToken: newRefreshToken,
          );

          // Повторяем исходный упавший запрос с новым токеном:
          final originalOptions = err.requestOptions;
          originalOptions.headers['Authorization'] = 'Bearer $newAccessToken';

          final cloneResponse = await _dio.fetch(originalOptions);
          return handler.resolve(cloneResponse);
        } catch (refreshErr) {
          // Если Refresh токен тоже протух: очищаем хранилище и выходим из профиля
          await _tokenStorage.clear();
        }
      }
    }

    handler.next(err);
  }
}
```

> [!NOTE] QueuedInterceptor
> Использование `QueuedInterceptor` гарантирует, что если 5 параллельных запросов одновременно получат 401, запрос на `/auth/refresh` выполнится **ровно один раз**. Остальные 4 запроса встанут в очередь и повторятся с полученным токеном.

---

## 4. Антипаттерны: как нельзя и как правильно

### ❌ Как нельзя:
```dart
// ОШИБКА 1: Ручное добавление заголовка в каждом запросе
final response = await http.get(
  Uri.parse('/tasks'),
  headers: {'Authorization': 'Bearer $myToken'}, // Код дублируется во всех 50 репозиториях!
);

// ОШИБКА 2: Отсутствие очереди при обновлении токена
// 10 одновременных запросов пошлют 10 вызовов refresh_token, заблокировав аккаунт

// ОШИБКА 3: Хранение пароля пользователя в открытом виде
await prefs.setString('password', pass); // Грубейшее нарушение безопасности!
```

### ✅ Как правильно:
```dart
// Централизованный AuthInterceptor прозрачно управляет токенами:
dio.interceptors.add(AuthInterceptor(...));

// Токены надежно изолированы в KeyStore/Keychain:
await secureStorage.write(key: 'access_token', value: token);

// Код репозитория чист и свободен от рутинной логики авторизации:
final response = await _dio.get('/api/v1/tasks');
```

---

## Резюме

1. Сетевое взаимодействие централизовано через клиент `Dio` и API Gateway.
2. Токены хранятся в зашифрованном виде через `flutter_secure_storage`.
3. `AuthInterceptor` прозрачно обновляет протухшие токены без прерывания пользовательского сеанса.
4. Архитектура надежно защищает данные студентов от перехвата и утечек.
