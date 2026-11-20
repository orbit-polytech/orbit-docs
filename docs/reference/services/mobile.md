# mobile (Мобильное приложение Flutter)

**Репозиторий:** [`orbit-polytech/mobile`](https://github.com/orbit-polytech/mobile)

**Статус:** работает сейчас (бета `0.1.16-beta+17`); собираются и публикуются релизные Android APK.

---

## 1. Назначение

Кроссплатформенный клиент Orbit для Android и iOS: расписание занятий с офлайн-кэшем, выбор и синхронизация учебных групп, авторизация по Email OTP и Google OAuth (с привязкой аккаунта), а также потоковый AI-ассистент с историей диалогов и оценками ответов.

---

## 2. Стек

- **Язык и SDK:** Dart `^3.13.7` / Flutter 3.x
- **Архитектура:** Clean Architecture + Feature-First (`app`, `core`, `design_system`, `features`)
- **Состояние:** `flutter_bloc ^9.1.1`, `bloc_concurrency`, `equatable`
- **Навигация:** `Navigator` + `IndexedStack` в `MainShellScreen` и `FloatingBottomNavBar` (в зависимостях также подключён `go_router ^18.1.2`)
- **Сеть:** `dio ^5.11.1` с интерцептором обновления токенов, SSE-стриминг AI
- **Хранилище:** `hive_ce` / `hive_ce_flutter` (кэш расписания, настройки, кэш истории AI) и `flutter_secure_storage ^11.2.0` (токены)
- **Авторизация и ссылки:** `app_links ^7.2.2`, `url_launcher ^6.3.1`, `crypto` (PKCE S256)
- **UI и утилиты:** `gpt_markdown`, `intl`, `package_info_plus`

---

## 3. Подключение к бэкенду

Мобильный клиент общается **только с `gateway-service` по HTTP/JSON и SSE**. Прямых обращений к gRPC или к `rasp.dmami.ru` из приложения нет.

Адрес шлюза передаётся при сборке/запуске через `--dart-define=BASE_URL=...` (без суффикса `/api/v1`; по умолчанию в debug используется `http://127.0.0.1:8080`).

Deep link схема для Google OAuth и привязки аккаунта на Android: `ru.mospolytech.orbit://auth/callback`.

---

## 4. Сборка, версионирование и проверки

```bash
# Зависимости
flutter pub get

# Обязательные проверки перед коммитом / PR
flutter analyze --fatal-infos
flutter test
python tool/check_version.py

# Запуск в debug с локальным Gateway на эмуляторе Android
flutter run --dart-define=BASE_URL=http://10.0.2.2:8080

# Запуск на телефоне по USB (после adb reverse tcp:8080 tcp:8080)
flutter run --dart-define=BASE_URL=http://127.0.0.1:8080
```

Перед открытием Pull Request с изменениями кода обязательно поднимайте в `pubspec.yaml` и видимую версию, и `build number` (например, `0.1.16-beta+17` → `0.1.17-beta+18`) — это проверяет CI через `tool/check_version.py`. Релизный workflow собирает APK с `--split-per-abi --obfuscate --split-debug-info=...`.
