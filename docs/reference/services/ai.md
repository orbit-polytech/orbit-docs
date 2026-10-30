# ai-service — потоковый AI и RAG

**Репозиторий:** [orbit-polytech/ai-service](https://github.com/orbit-polytech/ai-service). **Статус:** реализация в `main`, подключена к Gateway и профилям `ai` / `all` в infra.

## Назначение и границы

Генерация потоковых ответов, поиск по базе знаний, цитаты, сохранение результата и проверка владения диалогом. Сервис не выпускает JWT, не хранит профиль Users и пока не вызывает Schedule как инструмент.

## Текущий стек

| Слой | Реализация |
|---|---|
| Runtime | Python 3.13+, uv, `uv.lock` |
| Транспорт | `grpc.aio`, порт `50057`, health protocol |
| Генерация | Groq через OpenAI-совместимый SDK; модель задаётся `AI_MODEL` |
| Эмбеддинги | FastEmbed, `sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2`, 384 измерения |
| Хранилище | SQLAlchemy asyncio + asyncpg; PostgreSQL 16 + pgvector |
| Миграции | Alembic, отдельный `ai-migrate` |
| Контракты | Python-пакет `orbit-contracts` из Git по тегу `v0.5.2` |

База приложения — `orbit_ai`; изолированная база интеграционных тестов — `orbit_ai_test`. Между контейнерами используется `postgres:5432`, с хоста по умолчанию `localhost:5434`. Не переносите сюда старые `ai_knowledge_db`, `vector(768)` или Gemini env из учебного примера.

## Запрос и безопасность

Gateway проверяет Bearer JWT и передаёт `x-user-id` в gRPC metadata. AI требует идентичность и проверяет владельца. Этот заголовок сам по себе не является криптографической аутентификацией: прямой gRPC доступ должен быть ограничен доверенной сетью, локальный порт — loopback.

RPC: `ChatStream` и `CancelGeneration`. HTTP-клиент использует [AI API Gateway](../../gateway/ai-api.md), а не прямой порт сервиса.

## Конфигурация

Актуальный шаблон — [`.env.example`](https://github.com/orbit-polytech/ai-service/blob/main/.env.example). Основные параметры: `DATABASE_URL`, `GROQ_API_KEY`, `AI_MODEL`, `RAG_ENABLED`, `EMBEDDING_MODEL_NAME`, `RAG_MIN_SIMILARITY`, `RAG_TOP_K`, лимиты и таймауты `AI_*`. В `APP_ENV=production` отсутствие ключа или DSN останавливает запуск. В development без ключа используется FakeLLMClient — это не реальная генерация.

Настройки процесса и переменные, передаваемые Compose, различаются: произвольная строка в `infra/.env` не добавляет параметр контейнеру. См. [справочник env](../ports-and-env.md).

## Эксплуатация

В Docker модель эмбеддингов предварительно загружается при сборке; сервер проверяет БД и прогревает эмбеддер перед `SERVING`. Healthcheck не выполняет реальный запрос к Groq и не подтверждает качество базы знаний. Ingestion не запускается автоматически при старте сервера.

- [Запуск стенда с AI](../../how-to/ai-local-stack.md)
- [Поток обработки запроса](../../python-ai/request-flow.md)
- [Загрузка и версии документов](../../python-ai/knowledge-ingestion.md)
- [Ограничения](../../python-ai/current-limitations.md)
- [Диагностика](../../how-to/ai-diagnostics.md)

**Источники реализации:** [зависимости](https://github.com/orbit-polytech/ai-service/blob/main/pyproject.toml), [точка входа](https://github.com/orbit-polytech/ai-service/blob/main/src/orbit_ai/__main__.py), [Dockerfile](https://github.com/orbit-polytech/ai-service/blob/main/Dockerfile).
