# База знаний: загрузка и версии документов

Основной способ наполнения базы знаний — связка репозитория [`knowledge-pipeline`](../reference/services/knowledge-pipeline.md) (сбор и чистка документов в `manifest.json`) и команды загрузки в `ai-service`.

## Как устроено версионирование в PostgreSQL

`DocumentIngestor` в `ai-service` принимает документ (`source_id`, `title`, `url`, `domain`, `doc_type`, секции и контент), считает эмбеддинги FastEmbed и публикует версию в `orbit_ai`.

- Если составной хэш содержимого не изменился, документ пропускается без дубликатов.
- При обновлении текста старая версия деактивируется, а активной становится новая (защищено транзакцией, `advisory lock` по `source_id` и частичным уникальным индексом).
- RAG ищет только по активным версиям документов.

## Основной поток: `knowledge-pipeline` → `--manifest`

### Шаг 1. Собрать Knowledge Bundle в `knowledge-pipeline`

Из корня клона [`knowledge-pipeline`](https://github.com/orbit-polytech/knowledge-pipeline):

```bash
uv sync
uv run orbit-etl run --tiers 1,2
```

В результате появится каталог `output/` с `manifest.json`, очищенными `.md` / `.csv` / `.json` файлами, ручными справочниками из `curated_manual/` и отчётом `report.json`. Перед загрузкой в рабочую базу проверьте отчёт и содержимое материалов.

### Шаг 2. Загрузить манифест в `ai-service`

Сначала прогоните `--dry-run` (проверяет манифест и инициализирует FastEmbed без записи в БД), затем выполните загрузку.

**Локально из исходников `ai-service`** (укажите `DATABASE_URL` на хостовый `localhost:5434/orbit_ai`):

```bash
uv sync --locked
uv run python -m orbit_ai.scripts.ingest_documents --manifest ../knowledge-pipeline/output/manifest.json --dry-run
uv run python -m orbit_ai.scripts.ingest_documents --manifest ../knowledge-pipeline/output/manifest.json
```

**Или в работающий Docker-контейнер** (смонтировав каталог `knowledge-pipeline/output` как volume в `ai-service`):

```bash
docker compose --profile all run --rm \
  -v /path/to/knowledge-pipeline/output:/knowledge:ro \
  ai-service python -m orbit_ai.scripts.ingest_documents --manifest /knowledge/manifest.json
```

Готовый образ `ai-service` не содержит внутри каталог с документами: при запуске в контейнере каталог обязательно нужно примонтировать. Перед загрузкой убедитесь, что `ai-migrate` уже накатил миграции Alembic на `orbit_ai`.

---

## Запасной / legacy-вариант: встроенный краулер и `--dir`

Внутри `ai-service` остался старый встроенный краулер (`polytech_crawler`) и режим загрузки напрямую из папки Markdown (`--dir data/knowledge`). Для актуального корпуса лучше использовать `knowledge-pipeline` (он поддерживает PDF-регламенты, таблицы, `curated_manual/` и кэширование `ETag`), но для локальной проверки своих `.md` файлов `--dir` по-прежнему работает:

```bash
uv run python -m orbit_ai.scripts.ingest_documents --dir path/to/markdown --dry-run
uv run python -m orbit_ai.scripts.ingest_documents --dir path/to/markdown
```

## Чек-лист перед публикацией документов

1. Проверить первоисточник: актуальность редакции, даты и доступность URL.
2. Не менять `source_id` при обновлении уже загруженного документа, чтобы сработала замена версии, а не создался дубль.
3. Вычистить мусор навигации/шапок сайта и исключить персональные данные.
4. Использовать ту же модель эмбеддингов, что и в рантайме (`EMBEDDING_MODEL_NAME`, `384` измерения). Смена модели требует переиндексации.
5. После загрузки прогнать контрольные вопросы (или `run_eval`) и проверить цитаты.

## Тестирование ingestion

Интеграционные тесты запускайте только на отдельной тестовой базе `orbit_ai_test` через `POSTGRES_TEST_DSN` (после применения миграций к ней), не трогая рабочую `orbit_ai`.

**Источники:** [ingest_documents.py](https://github.com/orbit-polytech/ai-service/blob/main/src/orbit_ai/scripts/ingest_documents.py), [manifest_loader.py](https://github.com/orbit-polytech/ai-service/blob/main/src/orbit_ai/infrastructure/rag/ingestion/manifest_loader.py).
