# Flutter: Клиентское приложение Orbit

Репозиторий [**`mobile`**](https://github.com/orbit-polytech/mobile) — кроссплатформенное клиентское приложение Orbit для Android, iOS и **Web / PWA** (сейчас в статусе рабочей беты `v0.1.27-beta+28`).

---

## 1. Архитектура и текущий статус фич

Приложение построено по **Clean Architecture** и **Feature-First** (`Presentation -> Domain <- Data`), состояние экранов ведут `Bloc`/`Cubit` на `flutter_bloc 9.x` и `sealed classes`.

### Статусы функциональных блоков

* `[Реализовано]` — светлая дизайн-система (`AppTheme` и компоненты `design_system/`), адаптивный shell в `MainShellScreen`: нижняя панель `FloatingBottomNavBar` на мобильных экранах и боковая панель `DesktopShellSidebar` на планшетах/десктопе/вебе с тремя активными разделами (`Ассистент`, `Расписание`, `Профиль`).
* `[Реализовано]` — **Web / PWA-сборка**: контейнеризованная сборка Flutter Web (WASM) через `Dockerfile.web` (Nginx), манифест PWA `web/manifest.json`, адаптивное представление расписания `schedule_desktop_loaded_view.dart`, платформенная стратегия URL (`core/platform/url_strategy_*.dart`) и настройка HTTP-адаптера Dio для браузера и нативных платформ (`core/network/dio_adapter_*.dart`).
* `[Реализовано]` — расписание с **Offline-First** кэшем в `hive_ce`, поиском и переключением групп, избранными группами, календарём/неделями, обработкой `READY` / `PENDING_SYNC` / `is_stale` и шторкой деталей пары.
* `[Реализовано]` — гостевой режим и авторизация: **Email OTP** и **Google OAuth** с PKCE (и возвратом на мобильные устройства по deep link `ru.mospolytech.orbit://auth/callback`, а также web callback/cookie flow).
* `[Реализовано]` — синхронизация выбранных групп с сервером (`PreferencesSyncCoordinator`, `PATCH /api/v1/auth/preferences`) и разрешение конфликта при входе, а также экран настроек с привязкой и отвязкой Google.
* `[Реализовано]` — **Профиль пользователя (`features/profile`)**: загрузка и редактирование живого профиля через `users-service`, пресеты аватаров (`avatar_preset.dart` / `AvatarPresetPicker`), шторка редактирования профиля `EditProfileSheet`, смена почты `ChangeEmailSheet` и повторная аутентификация `ReauthSheet`.
* `[Реализовано]` — **ИИ-ассистент (`features/assistant`)**: потоковые ответы SSE, стадия поиска, отмена, история диалогов с поиском и группировкой по датам, удаление чатов, цитаты источников, выделение и цитирование фрагментов текста ответа, таймер обратного отсчёта при `429` (cooldown / rate limit), шторка расширенного фидбэка (`AssistantFeedbackBottomSheet`), оценки 👍/👎 с отправкой на бэкенд и повтор ответа.
* `[Реализовано]` — **Обращения в поддержку (`features/support`)**: отправка отзывов и баг-репортов через `ReportIssueBottomSheet`.
* `[Запланировано]` — раздел задач и дедлайнов (`tasks-service`), заметки и автоподтверждение академической группы/роли старосты через интеграцию с ЛК Политеха.

---

## 2. Взаимодействие слоёв

```text
┌─────────────────────────────────────────────────────────────┐
│                     PRESENTATION LAYER                      │
│   [ Screens / Views / Widgets ]  ◄── (State) ───┐           │
│               │                                 │           │
│               ▼ (Events / Calls)                │           │
│       [ BLoC / Cubit ] ─────────────────────────┘           │
└──────────────────────────┬──────────────────────────────────┘
                           │ UseCases / Repositories
┌──────────────────────────▼──────────────────────────────────┐
│                        DOMAIN LAYER                         │
│   [ UseCases / Entities / Failure / Repository Interfaces ] │
└──────────────────────────▲──────────────────────────────────┘
                           │ реализации контрактов
┌──────────────────────────┴──────────────────────────────────┐
│                         DATA LAYER                          │
│   [ RepositoryImpl ]                                        │
│       ├── LocalDataSource  ───► [ Hive CE / SecureStorage ] │
│       └── RemoteDataSource ───► [ Dio HTTP / SSE ] ─► GW    │
└─────────────────────────────────────────────────────────────┘
```

- **Presentation** отвечает только за UI и не ходит в сеть или базу напрямую.
- **Domain** не зависит от Flutter и внешних SDK.
- **Data** работает с локальным хранилищем (`HiveStorageService`, `SecureTokenStore`) и сетевым `DioClient` через Gateway.

---

## 3. Гостевой старт и вход в аккаунт

```text
               ┌──────────────────────────────────────┐
               │         Первый запуск Orbit          │
               └──────────────────┬───────────────────┘
                                  │
                                  ▼
               ┌──────────────────────────────────────┐
               │    Онбординг: выбор группы или вход  │
               └──────────────────┬───────────────────┘
                                  │
              ┌───────────────────┴───────────────────┐
              ▼                                       ▼
  [ Гостевой режим (Zero-Auth) ]         [ Вход по Email OTP / Google ]
  ├── Расписание группы и кэш Hive       ├── Всё из гостевого режима
  └── Избранные группы локально          ├── AI-ассистент, история и оценки
                                         ├── Синхронизация групп с аккаунтом
                                         └── Привязка Google в настройках
```

Без аккаунта студент сразу открывает расписание выбранной группы (в том числе без интернета, если неделя уже закэширована). Авторизация нужна только для персональных функций вроде AI-ассистента и облачной синхронизации настроек.

---

## 4. Фактическая структура `mobile/lib/`

```text
mobile/
├── Dockerfile.web                   # Сборка Web/WASM + Nginx для PWA
├── web/
│   ├── manifest.json                # Манифест PWA (Orbit)
│   └── nginx.conf
└── lib/
    ├── main.dart
    ├── app/
    │   ├── app.dart                 # OrbitApp, SessionBloc, PreferencesSyncCoordinator
    │   ├── app_providers.dart
    │   ├── bootstrap.dart
    │   ├── splash_screen.dart
    │   └── fallbacks/               # резервные заглушки репозиториев
    ├── core/
    │   ├── cancellation/            # CancellationToken (отмена операций)
    │   ├── config/app_config.dart   # BASE_URL через --dart-define, версия из package_info_plus
    │   ├── constants/               # ApiEndpoints, StorageKeys
    │   ├── extensions/
    │   ├── network/                 # DioClient, dio_adapter_*.dart, AuthInterceptor, ErrorHandler
    │   ├── platform/                # url_strategy_*.dart для Web/stub
    │   ├── services/                # DeepLinkService, ExternalBrowserLauncher и др.
    │   ├── storage/                 # HiveStorageService, AppCacheService
    │   ├── time/
    │   └── utils/
    ├── design_system/
    │   ├── components/              # AppButton, AppTextField, AppToast, AssistantAvatar, ErrorView, SkeletonPlaceholder
    │   ├── layout/                  # responsive_breakpoints.dart, метрики отступов
    │   ├── theme/app_theme.dart     # светлая тема AppTheme.lightTheme
    │   └── typography/              # AppTypography
    └── features/
        ├── assistant/               # ИИ-чат (SSE, история с поиском и группировкой, цитирование, 429 cooldown, фидбэк)
        ├── auth/                    # Email OTP, Google OAuth + PKCE, session, preferences sync, identity
        ├── profile/                 # профиль (users-service), пресеты аватаров, EditProfileSheet, ChangeEmailSheet, ReauthSheet
        ├── schedule/                # расписание (mobile + schedule_desktop_loaded_view.dart), группы, детали пары, Hive-кэш
        ├── shell/                   # MainShellScreen (IndexedStack), FloatingBottomNavBar, DesktopShellSidebar
        └── support/                 # обращения в поддержку (ReportIssueBottomSheet)
```

---

## 5. Дальше в этом разделе

* [**Архитектура BLoC и защита от гонок**](bloc-architecture.md) — состояния `ScheduleLoaded` / `SchedulePendingSync` / `ScheduleFailure`, `CancellationToken`, а также `AssistantBloc` и `SessionBloc`.
* [**Хранилище Hive, Cache-First и Сеть**](offline-and-storage.md) — реальные коробки `HiveStorageService`, `AuthInterceptor` с обновлением токенов и deep link OAuth.
* [**Дизайн-система: светлая тема и навигация**](design-tokens-and-ui.md) — цветовые роли, доступность и адаптивная навигация (`MainShellScreen`, `FloatingBottomNavBar`, `DesktopShellSidebar`).
* [**Групповые задачи старосты (Запланировано)**](group-tasks.md) — проектный набросок под будущие задачи и расширение ролевой модели.
