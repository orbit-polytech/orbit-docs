# Сеть, DTO и безопасное хранилище токенов

В мобильном приложении безопасность хранения пользовательских данных критична: студент не должен повторно авторизовываться при каждом открытии приложения, а украсть его JWT-токен из файловой системы устройства должно быть невозможно.

---

## 1. Безопасное хранилище токенов (`flutter_secure_storage`)

Для хранения JWT access-токена и 32-байтного refresh-токена используется библиотека `flutter_secure_storage`. Она использует аппаратные средства защиты операционных систем:
* **iOS**: Apple Keychain (с флагом `kSecAttrAccessibleAfterFirstUnlock`).
* **Android**: Android KeyStore с шифрованием AES/GCM.

```dart
// lib/data/services/secure_storage_service.dart
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class SecureStorageService {
  const SecureStorageService(this._storage);

  final FlutterSecureStorage _storage;

  static const _keyAccessToken = 'access_token';
  static const _keyRefreshToken = 'refresh_token';

  Future<void> saveTokens({required String accessToken, required String refreshToken}) async {
    await Future.wait([
      _storage.write(key: _keyAccessToken, value: accessToken),
      _storage.write(key: _keyRefreshToken, value: refreshToken),
    ]);
  }

  Future<String?> getAccessToken() => _storage.read(key: _keyAccessToken);
  Future<String?> getRefreshToken() => _storage.read(key: _keyRefreshToken);

  Future<void> clearTokens() => _storage.deleteAll();
}
```

---

## 2. HTTP-клиент с автоматическим Refresh токена

Клиент `ApiClient` оборачивает HTTP-вызовы к нашему `gateway-service` и автоматически перехватывает ошибку `401 Unauthorized`. Если 15-минутный access-токен истёк, клиент в фоне выполняет запрос на `/v1/auth/refresh`, обновляет токены и повторяет исходный запрос прозрачно для пользователя:

```dart
// lib/data/services/api/api_client.dart
import 'dart:convert';
import 'package:http/http.dart' as http;
import '../secure_storage_service.dart';

class ApiClient {
  ApiClient({required String baseUrl, required SecureStorageService storage})
      : _baseUrl = baseUrl,
        _storage = storage;

  final String _baseUrl;
  final SecureStorageService _storage;
  final http.Client _client = http.Client();

  Future<http.Response> get(String path) async {
    final token = await _storage.getAccessToken();
    final uri = Uri.parse('$_baseUrl$path');

    var response = await _client.get(
      uri,
      headers: {
        'Content-Type': 'application/json',
        if (token != null) 'Authorization': 'Bearer $token',
      },
    );

    // Если токен истёк (401), пробуем обновиться через Refresh-токен
    if (response.statusCode == 401) {
      final refreshed = await _refreshToken();
      if (refreshed) {
        final newToken = await _storage.getAccessToken();
        response = await _client.get(
          uri,
          headers: {
            'Content-Type': 'application/json',
            if (newToken != null) 'Authorization': 'Bearer $newToken',
          },
        );
      }
    }

    return response;
  }

  Future<bool> _refreshToken() async {
    final refreshToken = await _storage.getRefreshToken();
    if (refreshToken == null) return false;

    try {
      final resp = await _client.post(
        Uri.parse('$_baseUrl/v1/auth/refresh'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({'refresh_token': refreshToken}),
      );

      if (resp.statusCode == 200) {
        final data = jsonDecode(resp.body) as Map<String, dynamic>;
        await _storage.saveTokens(
          accessToken: data['access_token'] as String,
          refreshToken: data['refresh_token'] as String,
        );
        return true;
      }
    } catch (_) {
      // При ошибке очищаем сессию
    }

    await _storage.clearTokens();
    return false;
  }
}
```
