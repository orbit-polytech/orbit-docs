# knowledge-pipeline (ETL базы знаний Политеха)

**Репозиторий:** [`orbit-polytech/knowledge-pipeline`](https://github.com/orbit-polytech/knowledge-pipeline) (CLI `orbit-etl`).

**Статус:** работает сейчас. Используется для подготовки базы знаний, которую затем индексирует `ai-service`.

## Зачем вынесен в отдельный репозиторий

Сбор и чистка документов Московского Политеха отделены от рантайма `ai-service`:

- **`knowledge-pipeline` (`orbit-etl`)** обходит `mospolytech.ru` и официальные поддомены (`sitemap-files.xml` + PDF-регламенты), скачивает сырые файлы в `raw/` с HTTP-кэшированием (`ETag`, `sha256`), чистит HTML/PDF/таблицы, подмешивает вручную проверенные справочники (`curated_manual/`) и собирает стандартизированный Knowledge Bundle в `output/`.
- **`ai-service`** не тащит зависимости парсинга (`pymupdf`, `beautifulsoup4`, `lxml`), а занимается только онлайн-RAG и инкрементальной индексацией готового `output/manifest.json` командой `--manifest`.

## Стек и структура

- **Runtime:** Python 3.13+, `uv`, `uv.lock`
- **Зависимости:** `httpx`, `beautifulsoup4`, `lxml`, `pymupdf`, `pydantic`, `pyyaml`

Слои репозитория:

```text
knowledge-pipeline/
├── config/
│   ├── domains.yaml              # канонические домены и метаданные для RAG
│   ├── sources.yaml              # реестр источников (генерируется discover, можно править вручную)
│   ├── classification_rules.yaml # правила классификации документов по доменам
│   ├── extra_seeds.yaml          # дополнительные стартовые URL
│   ├── manual_sources.yaml       # внешние ручные источники
│   ├── parser_rules.yaml         # правила очистки HTML/блоков
│   └── pdf_hubs.yaml             # страницы со списками PDF-регламентов
├── curated_manual/               # вручную выверенные справочники (глоссарий, памятка ЛК/ЕУЗ/СДО)
├── raw/                          # локальный кэш сырых HTML/PDF (в .gitignore)
├── output/                       # итоговый bundle: manifest.json, .md/.csv/.json, report.json (в .gitignore)
└── src/orbit_etl/                # cli, config, discovery, fetchers, parsers, builder, common
```

## Основные команды

Из корня клона `knowledge-pipeline`:

```bash
uv sync

# 1. Найти актуальные страницы и PDF по sitemap и seed-спискам
uv run orbit-etl discover

# 2. Скачать источники уровней 1 и 2 с учётом ETag / sha256
uv run orbit-etl fetch --tiers 1,2

# 3. Очистить документы, объединить с curated_manual/ и собрать output/manifest.json
uv run orbit-etl build --tiers 1,2

# Либо выполнить весь цикл подряд:
uv run orbit-etl run --tiers 1,2
```

Как загрузить собранный `output/manifest.json` в базу `orbit_ai` (локально или через volume в контейнер `ai-service`), расписано в инструкции [База знаний и ingestion](../../python-ai/knowledge-ingestion.md).
