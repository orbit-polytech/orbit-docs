# Orbit AI — Инженерная база знаний и руководство разработчика

Добро пожаловать в официальную техническую документацию и обучающий портал распределенной системы **Orbit AI** — цифрового ассистента для студентов и преподавателей Московского Политеха.

Документация построена по международному архитектурному стандарту технической документации **[Diátaxis](https://diataxis.fr/)** и разделена на четыре взаимодополняющие части: обучающий курс, практические рецепты, архитектурные концепции и справочник.

---

## 1. Архитектурная схема экосистемы

Система Orbit AI состоит из 14 специализированных репозиториев, объединенных в единую организацию на GitHub: [`https://github.com/orbit-polytech`](https://github.com/orbit-polytech).

```mermaid
flowchart TD
    subgraph Clients ["Клиенты (Недоверенная зона)"]
        MOB["📱 client-mobile<br/>(Flutter 3.x / BLoC)"]
        BOT["🤖 bot-service<br/>(Python 3.12 / aiogram 3)"]
        WEB["🌐 Web / Swagger UI"]
    end

    subgraph Perimeter ["Периметр безопасности"]
        GW["🛡️ gateway-service (HTTP :8080)<br/>Валидация JWT Ed25519, Rate Limit, CORS"]
    end

    subgraph Mesh ["Внутренний доверенный контур (gRPC по HTTP/2)"]
        AUTH["auth-service<br/>(:50051)"]
        USERS["users-service<br/>(:50052)"]
        SCHED["schedule-service<br/>(:50053)"]
        NAV["navigation-service<br/>(:50054)"]
        TASKS["tasks-service<br/>(:50055)"]
        NOTIF["notification-service<br/>(:50056)"]
        AI["ai-service (Python / RAG)<br/>(:50057)"]
    end

    subgraph Storage ["Инфраструктура хранения (Database-per-Service)"]
        PG["🐘 PostgreSQL 16<br/>(tasks_db, schedule_db, ai_knowledge_db)"]
        REDIS["⚡ Redis 7<br/>(Кэш, Blacklist токенов)"]
        RMQ["🐇 RabbitMQ<br/>(Асинхронные события / Outbox)"]
    end

    MOB -->|REST JSON + JWT| GW
    BOT -->|REST JSON| GW
    WEB -->|REST JSON| GW

    GW ==>|gRPC + x-user-id| AUTH
    GW ==>|gRPC + x-user-id| USERS
    GW ==>|gRPC + x-user-id| SCHED
    GW ==>|gRPC + x-user-id| NAV
    GW ==>|gRPC + x-user-id| TASKS
    GW ==>|gRPC + x-user-id| AI

    TASKS -.->|События task.created| RMQ
    RMQ -.->|Потребление| NOTIF
    TASKS --> PG
    SCHED --> REDIS
    SCHED --> PG
    AI --> PG
```

---

## 2. Структура документации (Diátaxis Framework)

<div class="grid cards" markdown>

-   :mortar_board: __[Часть I. Учебный курс (Tutorial)](tutorial/common/00-setup.md)__
    
    ---
    
    Пошаговое практическое погружение с нуля. 4 специализированных трека:
    - **Общий:** Git, командный workflow, запуск `orbit-infra`, Protobuf Buf.
    - **Go Backend:** 13 глав от основ Go 1.24 до Clean Architecture, PostgreSQL, RabbitMQ и Docker.
    - **AI & Python:** gRPC-стек, RAG-поиск в `pgvector` по регламентам Политеха, Google Gemini 2.0.
    - **Mobile (Flutter):** Feature-First архитектура, BLoC, защищенный JWT-клиент.

-   :wrench: __[Часть II. Рецепты (How-to Guides)](how-to/create-service-from-template.md)__
    
    ---
    
    Пошаговые алгоритмы решения типовых инженерных задач:
    - Создание микросервиса из шаблона;
    - Добавление нового gRPC-метода и миграции базы данных;
    - Transactional Outbox в RabbitMQ и кэширование в Redis;
    - Интерактивная отладка в VS Code и решение конфликтов слияния в Git.

-   :bulb: __[Часть III. Архитектура и концепции (Explanation)](explanation/clean-architecture.md)__
    
    ---
    
    Глубокий теоретический разбор архитектурных решений:
    - Принципы Clean Architecture и Domain-Driven Design (DDD);
    - Почему gRPC внутри контура и REST снаружи;
    - Архитектура безопасности (JWT Ed25519 и метаданные);
    - 6 официальных архитектурных решений (**ADR-001 .. ADR-006**).

-   :books: __[Часть IV. Справочник (Reference)](reference/polyrepo-map.md)__
    
    ---
    
    Точные технические спецификации и таблицы:
    - Карта 14 репозиториев организации;
    - Сводная таблица портов и переменных окружения;
    - Спецификация кодов ошибок и маппинга HTTP/gRPC;
    - Документация REST API шлюза (OpenAPI / Swagger);
    - Паспорта и спецификации всех микросервисов.

</div>

---

## 3. Быстрый старт для нового разработчика

Если вы только присоединились к разработке Orbit AI, выполните следующие три шага:

1. **Подготовьте рабочее окружение:**  
   Изучите главу [**00. Подготовка рабочего места**](tutorial/common/00-setup.md) и установите компиляторы, Docker и расширения редактора.
2. **Запустите локальную инфраструктуру:**  
   Склонируйте репозиторий [`orbit-infra`](https://github.com/orbit-polytech/orbit-infra) и поднимите базы данных, Redis и RabbitMQ по руководству [**02. Запуск локального окружения**](tutorial/common/02-local-infra.md).
3. **Выберите свой трек разработки:**  
   Переходите к профильному треку ([Go Backend](tutorial/backend/01-go-essentials.md), [AI & Python](tutorial/ai/01-ai-grpc-stack.md) или [Mobile Flutter](tutorial/mobile/01-flutter-basics.md)).
