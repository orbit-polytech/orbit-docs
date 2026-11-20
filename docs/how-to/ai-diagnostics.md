# Диагностика AI и RAG

Для проверки полного пути с HTTP/SSE запускайте `infra` с профилем `all`:

```bash
docker compose --profile all ps -a
docker compose --profile all logs --tail 100 postgres-init ai-migrate ai-service gateway-service
```

Если включён стек мониторинга (`--profile monitoring` / `--profile all`), живые логи удобнее фильтровать в Grafana (`Orbit — Centralized Logs (Loki)`, `service="ai-service"`), а метрики смотреть на дашборде `Orbit — AI & LLM Tokens / Quotas` или в Prometheus (`http://127.0.0.1:9090`).

| Симптом | Что проверить |
|---|---|
| `ai-service` не поднялся после `task up` | `task up` включает только `apps`; нужен `--profile all` или `--profile ai` |
| Ошибка скачивания `ai-service` | Read-доступ к пакету `ghcr.io/orbit-polytech/ai-service` и свежий `docker login ghcr.io` |
| AI не стартует в `production` | Не задан API-ключ для выбранного `AI_PROVIDER` (`GROQ_API_KEY` / `OPENROUTER_API_KEY` / `GEMINI_API_KEY`) или неверный `DATABASE_URL` |
| Ошибка валидации `AI_MAX_OUTPUT_TOKENS` | Для провайдера Groq в коде стоит ограничение `<= 2048` |
| Мигратор `ai-migrate` упал | Образ `pgvector/pgvector:pg16`, прогон `task db:init`, права `orbit_user` на `orbit_ai` |
| Контейнер `healthy`, но генерация падает / `429` | Healthcheck не ходит во внешние LLM API; проверьте модель, ключи, fallback-цепочку и остаток квот (в Grafana / `/quotas` в `ops-bot`) |
| `401` / `403` | Действующий Access JWT, владелец диалога (передача чужого `conversation_id`) |
| `400` / `409` при повторе запроса | UUIDv4 формат `message_id`, совпадение payload (включая `model`), не завершившаяся или сбойная предыдущая попытка |
| Запрос уходит не в ту модель или пропускает RAG | `RAG_ENABLED`, `RAG_ROUTER_ENABLED`, логи `HybridQueryRouter`, `AI_ALLOWED_MODELS` |
| Ответ без цитат или пустой поиск | Загружен ли `manifest.json` через `ingest_documents`, активные версии документов, `RAG_MIN_SIMILARITY`, `RAG_TOP_K`, совпадение `EMBEDDING_MODEL_NAME` |
| Ответ приходит целиком в самом конце | Флаг `curl -N`, буферизация на внешнем прокси, SSE-парсер на клиенте |
| `cancelled=false` на `/chat/cancel` | Генерация уже завершилась, либо процесс перезапущен (реестр отмены in-memory на одну реплику) |
| Live-тесты не проходят | `POSTGRES_TEST_DSN`, отдельно применённые миграции к `orbit_ai_test` |

## Минимальный чек-лист приёмки

1. Чистый запуск с `--profile all` и перезапуск без потери томов.
2. Загрузка манифеста из `knowledge-pipeline` (`--manifest`) без дублей и с заменой старой версии документа.
3. Потоковый ответ с валидным JWT (`start` → при необходимости `retrieval` → `delta` → `citations` → `done` с `duration_ms`).
4. Проверка истории диалогов (`GET /conversations`, `GET /conversations/{id}`), удаления и отправки оценки (`PUT .../feedback`).
5. Проверка изоляции пользователей (`403` на чужом диалоге) и повтора (`replay`) без повторного вызова LLM.
6. Проверка поступления метрик в Prometheus (`ai-service:9093`), логов в Loki и дизлайков в `#feedback` у `ops-bot`.

Не отправляйте в чаты и PR реальные JWT, API-ключи, DSN с паролями и содержимое `.env`.
