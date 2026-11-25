# mobile (Клиент Flutter: Android и Web/PWA)

**Репозиторий:** [`orbit-polytech/mobile`](https://github.com/orbit-polytech/mobile)

**Статус:** работает сейчас (бета `0.1.27-beta+28`); собираются релизные Android APK и Docker-образ Web/PWA (`ghcr.io/orbit-polytech/web:latest`).

---

## 1. Назначение

Кроссплатформенный клиент Orbit для Android, iOS и Web/PWA: расписание занятий с офлайн-кэшем, выбор и синхронизация учебных групп, авторизация и смена email по OTP, Google OAuth с привязкой аккаунта, профиль пользователя (`users-service`), форма отправки обращений в поддержку (`ops-bot`) и потоковый AI-ассистент с цитированием фрагментов, историей диалогов, оценками и обработкой rate-limit cooldown.

---

## 2. Стек и модули

- **Язык и SDK:** Dart `^3.13.7` / Flutter 3.x (для Web WASM-сборки используется образ `ghcr.io/cirruslabs/flutter:3.47.9`)
- **Архитектура:** Clean Architecture + Feature-First (`app`, `core`, `design_system`, `features`: `assistant`, `auth`, `profile`, `schedule`, `shell`, `support`)
- **Состояние:** `flutter_bloc ^9.1.1`, `bloc_concurrency`, `equatable`
- **Навигация:** `Navigator` + `IndexedStack` в `MainShellScreen` и `FloatingBottomNavBar` (в зависимостях также подключён `go_router ^18.1.2`)
- **Сеть:** `dio ^5.11.1` с интерцептором обновления токенов, SSE-стриминг AI
- **Хранилище:** `hive_ce` / `hive_ce_flutter` (кэш расписания, настройки, кэш истории AI) и `flutter_secure_storage ^11.2.0` (токены)
- **Авторизация и ссылки:** `app_links ^7.2.2`, `url_launcher ^6.3.1`, `crypto` (PKCE S256)
- **UI и утилиты:** `gpt_markdown`, `intl`, `package_info_plus`

Ключевые функциональные модули:
- **`features/profile`**: просмотр и редактирование профиля (`GET`/`PUT /api/v1/users/profile`), пресеты аватарок, выбор группы/курса, смена email (`/auth/email/change/begin|confirm` через re-auth), привязка/отвязка Google.
- **`features/support`**: форма баг-репортов и обращений (`POST /api/v1/support/issues` по категориям `schedule`, `assistant`, `profile`, `ui_bug`, `other`), доступная и авторизованным пользователям, и гостям.
- **`features/assistant`**: SSE-стриминг, история и удаление диалогов, оценки, цитирование/ответ на фрагменты сообщений (`AssistantQuoteFormatter`, до 400 символов на цитату с возможностью комментария) и таймер обратного отсчёта `429` cooldown (`AssistantCooldownTimer` по `retry_after_sec` / `Retry-After`).

---

## 3. Подключение к бэкенду и Web/PWA

Клиент общается **только с `gateway-service` по HTTP/JSON и SSE**. Прямых обращений к gRPC или к `rasp.dmami.ru` из приложения нет.

- **Android / Desktop:** адрес шлюза передаётся через `--dart-define=BASE_URL=...` (без суффикса `/api/v1`; по умолчанию в debug используется `http://127.0.0.1:8080`). Deep link схема для Google OAuth и привязки аккаунта на Android: `ru.mospolytech.orbit://auth/callback`.
- **Web / PWA (`web` в `infra`):** собирается через `Dockerfile.web` (`flutter build web --release --wasm --no-web-resources-cdn` + `nginx:1.27-alpine`) с конфигом `web/nginx.conf`. Контейнер слушает `:80` внутри Compose и публикуется на хосте как **`127.0.0.1:8088`** (`WEB_HOST_PORT`). Nginx отдаёт сжатую статику с COOP/COEP-заголовками для WASM и проксирует `/api/`, `/healthz` и `/swagger/` на `http://gateway-service:8080` (с отключённой буферизацией для SSE и `proxy_read_timeout 180s`). При включённом профиле `tunnel` `ngrok` направляет трафик на `web:80`.

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

Перед открытием Pull Request с изменениями кода обязательно поднимайте в `pubspec.yaml` и видимую версию, и `build number` (например, `0.1.27-beta+28` → `0.1.28-beta+29`) — это проверяет CI через `tool/check_version.py`. Релизный workflow собирает Android APK с `--split-per-abi --obfuscate --split-debug-info=...`, а также публикует Web/PWA образ `web`.
