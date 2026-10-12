# Flutter: Мобильное приложение Orbit Mobile

Репозиторий [**`mobile`**](https://github.com/orbit-polytech/mobile) содержит исходный код кроссплатформенного клиентского приложения для студентов Московского Политеха, работающего на **Android** и **iOS**.

---

## 1. Официальная архитектура MVVM

Приложение построено по стандарту архитектуры **Model-View-ViewModel (MVVM)** в строгом соответствии с официальным руководством [Flutter App Architecture Guide](https://docs.flutter.dev/app-architecture) и образцовым проектом *Compass App* от команды Google Flutter.

### Схема взаимодействия слоёв:

```text
[ View (Экран / Виджет) ] 
       │ ▲ 
       │ │ ChangeNotifier / Command
       ▼ │ 
[ ViewModel ] 
       │ ▲ 
       │ │ Result<T>
       ▼ │ 
[ Repository (Data Layer) ] 
       │ ▲ 
       ▼ │ 
[ Service (HTTP ApiClient / SecureStorage) ] ──> HTTP ──> [ gateway-service ]
```

* **View**: отвечает исключительно за отрисовку пользовательского интерфейса и трансляцию действий пользователя (клики, ввод текста) во ViewModel.
* **ViewModel**: хранит состояние экрана (загрузка, ошибка, готовые данные), выполняет валидацию полей ввода и вызывает методы репозитория.
* **Repository**: единственный источник правды (Single Source of Truth) для конкретной предметной области. Преобразует сырые DTO из сети в неизменяемые доменные модели (`freezed`), кэширует данные в локальном хранилище.
* **Service**: низкоуровневый клиент, выполняющий один конкретный тип запросов (`ApiClient` к HTTP Gateway, `SecureStorageService` к зашифрованному хранилищу ключей).

---

## 2. Структура каталогов (`lib/`)

```text
mobile/
├── lib/
│   ├── config/
│   │   ├── dependencies.dart        # Регистрация Service Locator / провайдеров зависимостей
│   │   └── assets.dart              # Константы путей к иконкам и шрифтам
│   ├── data/                        # Слой данных (разложен по техническому типу)
│   │   ├── repositories/
│   │   │   ├── auth/
│   │   │   │   ├── auth_repository.dart          # Абстрактный интерфейс
│   │   │   │   ├── auth_repository_remote.dart   # Боевая реализация через Gateway
│   │   │   │   └── auth_repository_dev.dart      # Моковая реализация для быстрого UI-тестирования
│   │   │   └── tasks/
│   │   │       ├── task_repository.dart
│   │   │       ├── task_repository_remote.dart
│   │   │       └── task_repository_local.dart
│   │   └── services/
│   │       ├── api/
│   │       │   ├── api_client.dart               # Базовый HTTP-клиент с интерцепторами токенов
│   │       │   └── model/                        # DTO сетевых запросов и ответов
│   │       └── secure_storage_service.dart       # Хранение JWT в Keychain / KeyStore
│   ├── domain/                      # Доменные сущности приложения
│   │   └── models/
│   │       ├── task/
│   │       │   └── task.dart                     # Неизменяемая freezed-модель
│   │       └── user/
│   │           └── user.dart
│   ├── routing/
│   │   ├── router.dart              # Декларативная маршрутизация go_router
│   │   └── routes.dart              # Константы путей маршрутов
│   ├── ui/                          # Пользовательский интерфейс (разложен по фичам)
│   │   ├── core/                    # Общие темы оформления и глобальные виджеты
│   │   │   ├── themes/
│   │   │   └── ui/                  # Button, Card, ErrorIndicator
│   │   ├── auth/                    # Фича входа в систему
│   │   │   ├── view_models/
│   │   │   │   └── login_viewmodel.dart
│   │   │   └── widgets/
│   │   │       ├── login_screen.dart
│   │   │       └── telegram_button.dart
│   │   └── tasks/                   # Фича задач и дедлайнов
│   │       ├── view_models/
│   │       │   └── tasks_viewmodel.dart
│   │       └── widgets/
│   │           ├── tasks_screen.dart
│   │           └── task_tile.dart
│   ├── utils/
│   │   ├── command.dart             # Паттерн Command для вызова асинхронных действий
│   │   └── result.dart              # Sealed class Result<T>: Ok(value) / Error(exception)
│   ├── main.dart                    # Базовая точка входа
│   └── main_development.dart        # Запуск с dev-окружением и локальным gateway-service
├── test/                            # Модульные и виджет-тесты
├── pubspec.yaml
└── analysis_options.yaml
```

> [!NOTE]
> Слой `data` сгруппирован по техническим типам (`repositories`, `services`), так как один репозиторий используется сразу на многих экранах. Слой `ui`, напротив, разложен по фичам (`auth`, `tasks`, `schedule`), так как каждая пара View + ViewModel принадлежит конкретному пользовательскому экрану.
