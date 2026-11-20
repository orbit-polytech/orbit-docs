# Flutter: Мобильное приложение Orbit Mobile

Репозиторий [**`mobile`**](https://github.com/orbit-polytech/mobile) — клиентское приложение Orbit на Android и iOS (сейчас в статусе рабочей беты `0.1.16-beta+17`).

---

## 1. Архитектура и текущий статус фич

Приложение построено по **Clean Architecture** и **Feature-First** (`Presentation -> Domain <- Data`), состояние экранов ведут `Bloc`/`Cubit` на `flutter_bloc 9.x` и `sealed classes`.

### Статусы функциональных блоков

* `[Реализовано]` — светлая дизайн-система (`AppTheme` и компоненты `design_system/`) и нижняя навигационная панель `FloatingBottomNavBar` с тремя активными разделами (`Ассистент`, `Расписание`, `Профиль`).
* `[Реализовано]` — расписание с **Offline-First** кэшем в `hive_ce`, поиском и переключением групп, избранными группами, календарём/неделями, обработкой `READY` / `PENDING_SYNC` / `is_stale` и шторкой деталей пары.
* `[Реализовано]` — гостевой режим и авторизация: **Email OTP** и **Google OAuth** с PKCE и возвратом в приложение по deep link `ru.mospolytech.orbit://auth/callback`.
* `[Реализовано]` — синхронизация выбранных групп с сервером (`PreferencesSyncCoordinator`, `PATCH /api/v1/auth/preferences`) и разрешение конфликта при входе (`GroupConflictDialog`), а также экран настроек с привязкой и отвязкой Google.
* `[Реализовано]` — **ИИ-ассистент**: потоковые ответы SSE, стадия поиска, отмена, список и история диалогов, удаление чатов, цитаты источников, оценки 👍/👎 с отправкой на бэкенд и повтор ответа.
* `[Запланировано]` — раздел задач и дедлайнов (`tasks-service`), заметки и автоподтверждение группы/роли старосты через ЛК Политеха (`users-service`).

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
mobile/lib/
├── main.dart
├── app/
│   ├── app.dart                     # OrbitApp, SessionBloc, PreferencesSyncCoordinator
│   ├── app_providers.dart
│   ├── bootstrap.dart
│   ├── splash_screen.dart
│   └── fallbacks/                   # резервные заглушки репозиториев
├── core/
│   ├── cancellation/                # отмена операций
│   ├── config/app_config.dart       # BASE_URL через --dart-define, версия из package_info_plus
│   ├── constants/                   # endpoints API, ключи хранилища
│   ├── extensions/
│   ├── network/                     # DioClient, AuthInterceptor, ErrorHandler
│   ├── services/                    # ExternalBrowserLauncher и др.
│   ├── storage/hive_storage_service.dart
│   ├── time/
│   └── utils/
├── design_system/
│   ├── components/                  # AppButton, AppTextField, AssistantAvatar, ErrorView, SkeletonPlaceholder
│   ├── layout/                      # метрики отступов под FloatingBottomNavBar
│   ├── theme/app_theme.dart         # светлая тема AppTheme.lightTheme
│   └── typography/                  # AppTypography
└── features/
    ├── assistant/                   # ИИ-чат (SSE, история, удаление, фидбэк)
    ├── auth/                        # Email OTP, Google OAuth + PKCE, session, preferences sync, identity
    ├── profile/                     # профиль, выбор групп, SettingsScreen и привязка Google
    ├── schedule/                    # расписание, выбор недели, шторка деталей пары, Hive-кэш
    └── shell/                       # MainShellScreen (3 активные вкладки) и FloatingBottomNavBar
```

---

## 5. Дальше в этом разделе

* [**Архитектура BLoC и защита от гонок**](bloc-architecture.md) — состояния `ScheduleReady`, отмена запросов `CancelToken` и защита от гонки переключения групп.
* [**Хранилище Hive, Cache-First и Сеть**](offline-and-storage.md) — локальный кэш, `AuthInterceptor` с Single-Flight Refresh и deep link OAuth.
* [**Дизайн-система: светлая тема и навигация**](design-tokens-and-ui.md) — цветовые роли, доступность и навигация.
* [**Групповые задачи старосты (Запланировано)**](group-tasks.md) — проектный набросок под будущие `tasks-service` и `users-service`.
