# Orbit AI — инженерная база знаний

Orbit AI — экосистема сервисов для студентов Московского Политеха: авторизация, расписание, навигация, задачи и AI-консультант.

!!! info "Как читать эту документацию"
    Здесь одновременно описаны **текущая реализация**, **целевая архитектура** и **учебные примеры**. Будущие компоненты не считаются уже работающими. Начните со страницы [«Текущее состояние»](project/current-state.md).

<div class="grid cards" markdown>

-   :material-play-circle: __[Запустить актуальный стенд](quickstart/local-setup.md)__

    ---

    Поднять базовый стек Auth/Schedule/Gateway; отдельно подключить AI и pgvector через профиль `all`.

-   :material-progress-check: __[Узнать текущее состояние](project/current-state.md)__

    ---

    Что реализовано, что находится в разработке и какие компоненты относятся к целевой архитектуре.

-   :material-sitemap: __[Изучить целевую архитектуру](reference/polyrepo-map.md)__

    ---

    Карта полирепозитория, границы сервисов и план развития экосистемы.

</div>

## Расписание уже доступно

Публичный [поиск групп и просмотр расписания](gateway/schedule-api.md) работают без аккаунта. Запрос только с названием группы возвращает текущую неделю по Москве. Для полного стенда достаточно клонировать `infra`; исходники всех сервисов не нужны.

## Целевая архитектура

Целевая экосистема включает 13 специализированных репозиториев, объединённых API Gateway, Protobuf-контрактами и, на следующих этапах, асинхронной шиной событий.

![Архитектурная схема экосистемы Orbit AI](image/architecture-overview.png)

Схема показывает **целевое состояние**, не конфигурацию портов: старые подписи на иллюстрации могут отличаться от [канонического справочника](reference/canonical-values.md); фактический порт Schedule — `50054`. Реализованный срез перечислен на странице [текущего состояния](project/current-state.md).

## Разделы базы знаний

<div class="grid cards" markdown>

-   :material-rocket-launch: __[Быстрый старт](quickstart/overview.md)__

    Запуск стенда, правила команды, GitHub Flow и проверка Pull Request.

-   :simple-go: __[Go-бэкенд](backend-go/overview.md)__

    Clean Architecture, DDD, gRPC, PostgreSQL, тестирование и CI/CD.

-   :material-shield-check: __[HTTP Gateway](gateway/overview.md)__

    Публичный HTTP-периметр, middleware и gRPC-клиенты.

-   :simple-python: __[Python & AI](python-ai/overview.md)__

    Текущий Groq/FastEmbed/RAG, потоковый AI API и отдельно учебные варианты Gemini/Telegram.

-   :simple-flutter: __[Flutter & Мобилка](mobile-flutter/overview.md)__

    Целевая архитектура клиента, состояние UI и безопасная работа с токенами.

-   :material-transit-connection-variant: __[Контракты и инфраструктура](contracts-infra/overview.md)__

    API-first, Buf, Docker Compose и локальные зависимости.

-   :material-lock-check: __[Безопасность и Auth](security/overview.md)__

    Email OTP, Google OAuth, JWT Ed25519, refresh-сессии и будущие способы входа.

-   :material-book-open-page-variant: __[Справочник](reference/canonical-values.md)__

    Порты, переменные окружения, паспорта сервисов и ADR.

-   :material-robot: __[Для ИИ-агентов](ai/agents.md)__

    Правила работы с кодом и машиночитаемый индекс `llms.txt`.

</div>


## AI уже подключён

AI доступен авторизованным пользователям через потоковый [AI API](gateway/ai-api.md). Для запуска нужен [AI-профиль](how-to/ai-local-stack.md); обычный `task up` его не включает. База знаний наполняется отдельной операцией. См. [текущие ограничения](python-ai/current-limitations.md): наличие ответа не гарантирует подтверждение источниками.
