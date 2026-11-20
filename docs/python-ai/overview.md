# Python & AI: текущая реализация и учебные примеры

Рабочий `ai-service` работает на Python 3.13+, `uv`, `grpc.aio`, мультипровайдерном LLM-роутере (`groq`, `openrouter`, `gemini`), FastEmbed и PostgreSQL/`pgvector`. В отдельный репозиторий вынесен [`knowledge-pipeline`](../reference/services/knowledge-pipeline.md) (`orbit-etl`), а для команды разработки работает приватный админский [`ops-bot`](../reference/services/ops-bot.md). Пользовательский Telegram-бот `bot-service` пока остаётся в планах.

## Текущая структура `ai-service`

```text
ai-service/
├── src/orbit_ai/
│   ├── __main__.py              # точка входа
│   ├── bootstrap/               # container, lifecycle, llm_factory (Groq/OpenRouter/Gemini), OpenRouter poller
│   ├── config/                  # pydantic-settings, llm_defaults, model_policy, model_routing.json
│   ├── domain/                  # сущности, enum стадий/ошибок, порты
│   ├── application/
│   │   ├── chat_stream/         # pipeline, query_classifier, context_collector, context_budget, prompt_builder, citations, error_mapper
│   │   ├── feedback/            # submit_feedback
│   │   ├── cancel_generation_use_case.py
│   │   └── manage_conversations_use_case.py
│   ├── infrastructure/
│   │   ├── llm/                 # RouterLLMClient, HybridQueryRouter, rate_limit_hook, groq/, fake_client
│   │   ├── rag/                 # embeddings/ (FastEmbed), ingestion/ (manifest + md), retrieval/ (hybrid FTS+vector + reranker)
│   │   ├── storage/             # postgres/ (SQLAlchemy + Alembic), in_memory/, summary_builder
│   │   ├── telemetry/           # contextvars расхода LLM и Prometheus-метрики :9093
│   │   └── crawler/             # встроенный краулер (legacy/вспомогательный)
│   ├── presentation/grpc/       # servicer, auth metadata, mappers, server, health_probe
│   └── scripts/                 # ingest_documents (--manifest / --dir), eval_harness, run_eval, chat, polytech_crawler
├── migrations/versions/         # Alembic 001..006
├── tests/                       # unit, transport, integration
├── pyproject.toml
├── uv.lock
└── Dockerfile
```

Слой `domain` не зависит от транспорта и провайдеров, `application` оркестрирует сценарии, `infrastructure` отвечает за LLM/RAG/БД/метрики, `presentation` держит gRPC.

- [Паспорт ai-service](../reference/services/ai.md)
- [Паспорт knowledge-pipeline](../reference/services/knowledge-pipeline.md)
- [Поток AI-запроса](request-flow.md)
- [База знаний и ingestion](knowledge-ingestion.md)
- [Ограничения](current-limitations.md)
- [AI HTTP/SSE API](../gateway/ai-api.md)
- [Запуск стенда с AI](../how-to/ai-local-stack.md)

**Исходники:** [src/orbit_ai](https://github.com/orbit-polytech/ai-service/tree/main/src/orbit_ai).

---

!!! warning "Ниже сохранён исторический учебный пример"
    Разделы ниже (Python 3.12, структура `app/adapters/transport`, `google-genai`, `psycopg`) оставлены как учебный пример для сравнения архитектур. Реальный код в `main` устроен по схеме выше.

# Исторический учебный вариант Python/AI

В исходном учебном варианте стек Python 3.12 рассматривался для двух задач:
1. `ai-service` — RAG с Google Gemini и `psycopg 3`.
2. `bot-service` — Telegram-бот на `aiogram 3.x`.

## 1. Общие стандарты разработки на Python

* **Пакетный менеджер:** [**uv**](https://github.com/astral-sh/uv), детерминированный `uv.lock` без `requirements.txt`.
* **Линтинг и стиль:** **Ruff** (`uv run ruff check .`, `uv run ruff format --check .`).
* **Статическая типизация:** аннотации типов и `uv run mypy src` (`strict = true`).
* **Конфигурация:** `pydantic-settings`.
* **Логирование:** структурированный JSON в production (без `print()`).
* **Тестирование:** `pytest` и `pytest-asyncio`.

## 2. Учебная структура (`src`-layout)

```text
ai-service/
├── src/
│   └── orbit_ai/
│       ├── __init__.py
│       ├── __main__.py
│       ├── config.py
│       ├── domain/
│       ├── app/
│       ├── adapters/
│       └── transport/
├── tests/
├── pyproject.toml
└── uv.lock
```

## 3. Пример `pyproject.toml` (учебный)

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
```
