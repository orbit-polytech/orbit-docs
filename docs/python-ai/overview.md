# Python & AI: текущая реализация и учебные примеры

Рабочий `ai-service` использует Python 3.13+, uv, grpc.aio, Groq, FastEmbed и PostgreSQL/pgvector. Telegram `bot-service` остаётся целевой интеграцией, не запущенным компонентом стенда.

## Текущая структура ai-service

```text
ai-service/
├── src/orbit_ai/
│   ├── __main__.py           # сборка зависимостей, warmup и lifecycle
│   ├── config/              # pydantic-settings
│   ├── domain/              # сущности, состояния, порты
│   ├── application/         # chat stream и отмена
│   ├── infrastructure/      # Groq, FastEmbed, RAG, PostgreSQL
│   ├── presentation/grpc/   # servicer и health
│   └── scripts/             # ingestion demo
├── migrations/              # Alembic
├── tests/                   # unit, transport, integration
├── pyproject.toml
├── uv.lock
└── Dockerfile
```

Domain не зависит от транспорта/провайдера. Application управляет сценарием через порты, Infrastructure реализует внешние зависимости, Presentation преобразует gRPC. Точка входа собирает эти слои. Структура не требует размещать бизнес-правила в gRPC servicer.

- [Паспорт сервиса](../reference/services/ai.md)
- [Поток запроса](request-flow.md)
- [База знаний и ingestion](knowledge-ingestion.md)
- [Ограничения](current-limitations.md)
- [AI HTTP/SSE API](../gateway/ai-api.md)
- [Запуск в Docker](../how-to/ai-local-stack.md)

**Исходное дерево:** [src/orbit_ai](https://github.com/orbit-polytech/ai-service/tree/main/src/orbit_ai).

---

!!! warning "Сохранённый учебный пример — не текущее устройство сервиса"
    Ниже сохранены исходная структура и примеры Gemini/Python 3.12 для обучения и сравнения архитектур. Пути `app/adapters/transport`, зависимости, требования к логам и env этого примера не описывают текущий `main`. Актуальный код использует `application/infrastructure/presentation`, Python 3.13 и Groq/FastEmbed. Не копируйте старый pyproject поверх рабочего.

# Исторический учебный вариант Python/AI

В исходном учебном варианте стек Python 3.12 рассматривался для двух специализированных задач:
1. [**`ai-service`**](https://github.com/orbit-polytech/ai-service) — микросервис искусственного интеллекта, реализующий Retrieval-Augmented Generation (RAG), генерацию эмбеддингов через Google Gemini API и векторный поиск ближайших соседей в PostgreSQL (`pgvector`).
2. [**`bot-service`**](https://github.com/orbit-polytech/bot-service) — официальный Telegram-бот для студентов Политеха на базе асинхронного фреймворка `aiogram 3.x`.

Остальные микросервисы (расписание, задачи, авторизация, навигация) пишутся строго на Go.

---

## 1. Общие стандарты разработки на Python

* **Версия**: строго Python 3.12+.
* **Пакетный менеджер**: [**uv**](https://github.com/astral-sh/uv). Все зависимости фиксируются в `pyproject.toml` и детерминированном файле блокировки `uv.lock`. Традиционные файлы `requirements.txt` не используются.
* **Стиль и форматирование**: стандарт PEP 8, проверка и автоформатирование выполняются через ультрабыстрый линтер **Ruff** (`ruff check`, `ruff format`).
* **Статическая типизация**: строгие аннотации типов для всех функций и методов с обязательной проверкой `mypy --strict`.
* **Конфигурация**: строго через `pydantic-settings` с загрузкой из переменных окружения.
* **Логирование**: только структурированный JSON через `structlog` или `python-json-logger`. Вызовы `print()` в коде сервисов категорически запрещены.
* **Тестирование**: фреймворк `pytest`, для асинхронных тестов — `pytest-asyncio`.

---

## 2. Структура `ai-service` (Src-Layout)

Сервис организован по стандартной `src`-раскладке из официального руководства Python Packaging User Guide:

```text
ai-service/
├── src/
│   └── orbit_ai/
│       ├── __init__.py
│       ├── __main__.py          # Запуск gRPC-сервера (порт :50057)
│       ├── config.py            # Настройки Settings(BaseSettings)
│       ├── domain/
│       │   ├── __init__.py
│       │   └── models.py        # dataclass'ы: Document, Chunk, Answer
│       ├── app/
│       │   ├── __init__.py
│       │   ├── ports.py         # typing.Protocol: LLMClient, Embedder, VectorStore
│       │   └── ask.py           # Сценарий «Ответить на вопрос по базе знаний Политеха»
│       ├── adapters/
│       │   ├── __init__.py
│       │   ├── gemini.py        # Интеграция с Google Gemini API
│       │   └── pgvector.py      # Реализация векторного поиска через psycopg 3
│       └── transport/
│           ├── __init__.py
│           └── grpc_server.py   # gRPC Servicer (proto <-> domain)
├── tests/
│   ├── conftest.py
│   ├── test_ask.py
│   └── test_pgvector.py
├── pyproject.toml
├── uv.lock
├── .env.example
├── Dockerfile
└── Makefile
```

---

## 3. Зависимости в `pyproject.toml`

```toml
[project]
name = "orbit-ai"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = [
    "grpcio>=1.66",
    "google-genai>=1.0",
    "psycopg[binary,pool]>=3.2",
    "pgvector>=0.3",
    "pydantic-settings>=2.4",
]

[dependency-groups]
dev = [
    "pytest>=8.0",
    "pytest-asyncio>=0.23",
    "ruff>=0.6",
    "mypy>=1.11",
]

[tool.ruff]
line-length = 100
target-version = "py312"

[tool.mypy]
strict = true
```
