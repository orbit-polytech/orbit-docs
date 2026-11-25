# ai-service — потоковый AI, RAG и диалоги

**Репозиторий:** [`orbit-polytech/ai-service`](https://github.com/orbit-polytech/ai-service)

**Статус:** работает сейчас в `main` (`contracts@v0.5.10`), запускается в Compose через профили `ai` и `all`.

## Назначение и границы

Отвечает за потоковые ответы ИИ-ассистента, классификацию и маршрутизацию запросов, гибридный RAG-поиск по базе знаний, хранение диалогов, фидбэк пользователей и учёт токенов/квот. Сервис не выпускает JWT, не хранит профиль студента и пока не вызывает `schedule-service` как инструмент.

## Стек

| Слой | Реализация |
|---|---|
| Runtime | Python 3.13+, `uv`, `uv.lock` |
| Транспорт | `grpc.aio`, порт `50057`, gRPC Health Checking |
| Генерация LLM | Мультипровайдерный `RouterLLMClient` (Groq / OpenRouter / Gemini), роутинг ролей по `src/orbit_ai/config/model_routing.json` (`router`, `casual`, `knowledge_rag`, `campus_navigation`, `general_study`, `default_fallbacks`) |
| Эмбеддинги | FastEmbed, `sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2` (384 измерения) |
| Reranker | Опциональный cross-encoder (`RAG_RERANKER_ENABLED`, по умолчанию `BAAI/bge-reranker-v2-m3`) |
| Хранилище | SQLAlchemy asyncio + asyncpg; PostgreSQL 16 + `pgvector` (HNSW + Russian/simple FTS) |
| Миграции | Alembic `001–006` (отдельный one-shot контейнер `ai-migrate`) |
| Метрики | `prometheus-client` на внутреннем порту `:9093` (`/metrics`), учёт токенов и квот |
| Контракты | Пакет `orbit-contracts` из Git по тегу `v0.5.10` |

База приложения — `orbit_ai`, отдельная тестовая база — `orbit_ai_test`. В `production` логи пишутся в JSON (`service="ai-service"`).

## Безопасность и RPC

Gateway проверяет Bearer JWT и передаёт `x-user-id` в gRPC metadata (при настройке `ORBIT_INTERNAL_SERVICE_SECRET` дополнительно проверяется общий внутренний секрет). AI проверяет владение диалогом, повтором (replay), отменой и оценкой (при чужом ресурсе возвращает `PERMISSION_DENIED`).

Реализованные RPC в `orbit.ai.v1.AIService`:
- `ChatStream` (server streaming со стадиями `START`, `RETRIEVAL`, `DELTA`, `CITATIONS`, `DONE`, `CANCELLED`, `ERROR`);
- `CancelGeneration`;
- `GetConversations`, `GetConversationHistory`, `DeleteConversation`;
- `SubmitFeedback`.

Внешние клиенты работают через [AI API Gateway](../../gateway/ai-api.md), а не напрямую по gRPC.

## Конфигурация

Шаблон настроек: [`.env.example`](https://github.com/orbit-polytech/ai-service/blob/main/.env.example). Основные переменные: `DATABASE_URL`, `AI_PROVIDER` (`groq`/`openrouter`/`gemini`), `GROQ_API_KEY`, `OPENROUTER_API_KEY`, `GEMINI_API_KEY`, `AI_MODEL` (в коде `settings.py` по умолчанию `openai/gpt-oss-120b`, в `infra/.env.example` — `qwen/qwen3.8-27b`; при пустом `model` или `model: "auto"` в запросе используется роутинг ролей из `model_routing.json`), `AI_ALLOWED_MODELS`, `CHAT_HISTORY_LIMIT`, `RAG_ENABLED`, `RAG_ROUTER_ENABLED`, `RAG_RERANKER_ENABLED`, `RAG_MIN_SIMILARITY` (`0.45`), `RAG_TOP_K` (`8`), `METRICS_PORT` (`9093`), лимиты и таймауты `AI_*`.

В `APP_ENV=production` сервис не запустится без API-ключа для выбранного `AI_PROVIDER` и `DATABASE_URL`. Для Groq валидатор дополнительно ограничивает `AI_MAX_OUTPUT_TOKENS <= 2048`. В `development` без ключей поднимается `FakeLLMClient` для локальных проверок.

Помните, что Compose передаёт контейнеру только переменные, перечисленные в `compose.yaml` (см. [порты и env](../ports-and-env.md)).

## Эксплуатация и база знаний

- В Docker модель эмбеддингов FastEmbed загружается на этапе сборки образа; перед переходом в `SERVING` сервер проверяет БД и прогревает модель (и reranker, если включён).
- Актуальный корпус Московского Политеха собирается отдельным репозиторием [`knowledge-pipeline`](knowledge-pipeline.md) в `output/manifest.json` и загружается командой `ingest_documents --manifest`.
- Ошибки генерации и негативные отзывы студентов попадают из `orbit_ai` в топики `#untriaged` и `#feedback` операционного бота [`ops-bot`](ops-bot.md).

Дальше: [запуск стенда с AI](../../how-to/ai-local-stack.md), [поток запроса](../../python-ai/request-flow.md), [загрузка базы знаний](../../python-ai/knowledge-ingestion.md), [ограничения](../../python-ai/current-limitations.md), [диагностика](../../how-to/ai-diagnostics.md).
