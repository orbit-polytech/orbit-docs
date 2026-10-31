# Безопасное хранилище токенов и Сетевой клиент

> **Канонический стандарт проекта:**  
> Полная спецификация сетевого взаимодействия, реализации Single-Flight Refresh в `Dio` и кэширования в Hive зафиксирована в основном документе:  
> 📖 [**Хранилище Hive, Cache-First и Сетевой слой**](offline-and-storage.md).

В этом разделе рассматриваются механизмы аппаратной защиты токенов сессии и принципы работы с сетевыми DTO.

---

## 1. Аппаратное хранилище токенов (`FlutterSecureStorage`)

Для долговременного хранения Access и Refresh токенов в мобильном клиенте используется библиотека **`flutter_secure_storage`**. 

Она опирается на системные защищенные хранилища мобильных ОС:
* **iOS**: **Keychain Services** (с флагом `kSecAttrAccessibleAfterFirstUnlock` — данные доступны только после первой разблокировки устройства пользователем).
* **Android**: **Android KeyStore** с аппаратным шифрованием AES/GCM (ключи шифрования хранятся в изолированном TEE / StrongBox модуле процессора).

```dart
// lib/core/storage/secure_storage_service.dart
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class SecureStorageService {
  const SecureStorageService({FlutterSecureStorage? storage})
      : _storage = storage ??
            const FlutterSecureStorage(
              aOptions: AndroidOptions(encryptedSharedPreferences: true),
              iOptions: IOSOptions(accessibility: KeychainAccessibility.first_unlock),
            );

  final FlutterSecureStorage _storage;

  static const _keyAccessToken = 'orbit_access_token';
  static const _keyRefreshToken = 'orbit_refresh_token';

  Future<void> saveTokens({required String accessToken, required String refreshToken}) async {
    await Future.wait([
      _storage.write(key: _keyAccessToken, value: accessToken),
      _storage.write(key: _keyRefreshToken, value: refreshToken),
    ]);
  }

  Future<String?> getAccessToken() => _storage.read(key: _keyAccessToken);
  Future<String?> getRefreshToken() => _storage.read(key: _keyRefreshToken);

  Future<void> clearTokens() async {
    await Future.wait([
      _storage.delete(key: _keyAccessToken),
      _storage.delete(key: _keyRefreshToken),
    ]);
  }
}
```

---

## 2. Архитектура сетевого клиента (Dio)

В проекте Orbit Mobile в качестве основного HTTP-клиента используется **`Dio`**. 

Ключевые требования к сетевому слою:
1. **Единый шлюз:** Мобильное приложение взаимодействует с бэкендом исключительно через API Gateway (`/api/v1/...`).
2. **Изоляция публичных запросов:** Запросы расписания выполняются без авторизационных заголовков.
3. **Single-Flight Refresh:** Параллельные 401-ошибки объединяются в один запрос обновления токенов с мьютексом.
4. **Конфиденциальность логов:** Запрещен вывод заголовков `Authorization` и тел личных сообщений в логи.

Подробная реализация канонического `AuthInterceptor` с исходным кодом приведена в разделе [Хранилище Hive, Cache-First и Сетевой слой](offline-and-storage.md).
