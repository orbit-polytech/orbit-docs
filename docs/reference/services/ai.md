# ai-service (AI RAG и Gemini Сервис)

**Репозиторий:** [`orbit-polytech/ai-service`](https://github.com/orbit-polytech/ai-service)

---

## 1. Назначение
Сервис искусственного интеллекта на Python. Реализует пайплайн генеративно-поисковой модели (RAG — Retrieval-Augmented Generation) по официальным регламентам Московского Политеха (положения о БРС, стипендиях, общежитиях, справках военкомата, переводе на бюджет). Находит релевантные фрагменты через векторный поиск в `pgvector` и формирует точный ответ с цитированием документов через Google Gemini API.

---

## 2. Технический стек
- **Язык и рантайм:** Python 3.12+ (uv / pip)
- **gRPC сервер:** `grpcio` (асинхронный сервер `grpc.aio`)
- **База данных и векторный индекс:** PostgreSQL 16 + расширение `pgvector`, индекс `HNSW` (`vector_cosine_ops`), размерность `vector(768)`
- **Эмбеддинги:** Google `text-embedding-004` через SDK `google-genai`
- **Генеративная LLM:** Google `gemini-2.0-flash`
- **Драйвер БД:** `psycopg` (v3) с пулом соединений и регистрацией векторов

---

## 3. Выделенные порты и база данных
- **gRPC порт:** `50057`
- **База данных:** `ai_knowledge_db` в PostgreSQL
- **Таблица документов:** `knowledge_chunks` с полем `embedding vector(768)`

---

## 4. Обязательные переменные окружения

```ini
APP_ENV=local
LOG_LEVEL=info
GRPC_PORT=50057

# Подключение к PostgreSQL с pgvector
DB_HOST=localhost
DB_PORT=5432
DB_USER=orbit
DB_PASSWORD=orbit_dev_secret
DB_NAME=ai_knowledge_db
DB_SSLMODE=disable

# Google Gemini API
GEMINI_API_KEY="AIzaSy..."
GEMINI_MODEL="gemini-2.0-flash"
EMBEDDING_MODEL="text-embedding-004"
```
