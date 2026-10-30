# Альтернативный учебный RAG: Gemini и pgvector

!!! warning "Учебный / целевой вариант"
    Примеры ниже сохранены для обучения и не являются инструкцией к текущему стенду. Gemini, старые env и схема БД относятся к альтернативному варианту; Telegram-бот ещё не реализован. Актуальная реализация — [паспорт ai-service](../reference/services/ai.md).

В этом руководстве пошагово объясняется, как устроен пайплайн семантического поиска и генерации ответов (Retrieval-Augmented Generation, RAG) в `ai-service`.

---

## 1. Размер вектора в этом альтернативном примере: 768 float

В этом учебном Gemini-варианте выбрана размерность **`vector(768)`**. В текущем ai-service FastEmbed использует 384 измерения; нельзя смешивать эти схемы.

Почему именно 768?
* Это стандартная размерность актуальных эмбеддинг-моделей Google (`text-embedding-004`).
* Вектор из 768 чисел с плавающей точкой занимает всего 3 КБ в памяти, что позволяет строить быстрые графовые индексы **HNSW** в PostgreSQL и обслуживать поисковые запросы за 5–10 миллисекунд.

---

## 2. Схема базы данных и индекс HNSW

В базе данных PostgreSQL активируется расширение `vector`:

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE knowledge_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_title VARCHAR(255) NOT NULL,
    chunk_text TEXT NOT NULL,
    embedding vector(768) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Графовый индекс HNSW для косинусного расстояния (<=>)
CREATE INDEX idx_knowledge_chunks_embedding 
ON knowledge_chunks 
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);
```

> [!TIP]
> Индекс **HNSW (Hierarchical Navigable Small World)** значительно превосходит старый индекс `ivfflat`: он не требует предварительного обучения на данных, устойчив к росту базы знаний и обеспечивает высочайшую точность поиска (Recall > 98%).

---

## 3. Генерация эмбеддингов через Google GenAI SDK

Для работы с Gemini используется современный единый SDK `google-genai`:

```python
import math
from google import genai
from google.genai import types

def normalize_vector(v: list[float]) -> list[float]:
    """Нормализует вектор к единичной длине L2 для быстрого косинусного поиска."""
    norm = math.sqrt(sum(x * x for x in v))
    if norm == 0.0:
        return v
    return [x / norm for x in v]

class GeminiEmbedder:
    def __init__(self, client: genai.Client, model: str, dim: int = 768) -> None:
        self._client = client
        self._model = model
        self._dim = dim

    def embed_texts(self, texts: list[str]) -> list[list[float]]:
        response = self._client.models.embed_content(
            model=self._model,
            contents=texts,
            config=types.EmbedContentConfig(output_dimensionality=self._dim),
        )
        return [normalize_vector(e.values) for e in response.embeddings]
```

---

## 4. Поиск ближайших контекстов в PostgreSQL (`psycopg 3`)

```python
import psycopg
from pgvector.psycopg import register_vector

class PgVectorStore:
    def __init__(self, db_url: str) -> None:
        self._db_url = db_url

    def search_similar(self, query_vector: list[float], limit: int = 3) -> list[str]:
        with psycopg.connect(self._db_url) as conn:
            register_vector(conn)
            with conn.cursor() as cur:
                # Оператор <=> вычисляет косинусное расстояние
                cur.execute(
                    """
                    SELECT chunk_text
                    FROM knowledge_chunks
                    ORDER BY embedding <=> %s
                    LIMIT %s;
                    """,
                    (query_vector, limit),
                )
                rows = cur.fetchall()
                return [row[0] for row in rows]
```

---

## 5. Генерация финального ответа через LLM

Найденные фрагменты знаний подставляются в системный промпт модели:

```python
class RAGUseCase:
    def __init__(self, embedder: GeminiEmbedder, store: PgVectorStore, client: genai.Client, chat_model: str) -> None:
        self._embedder = embedder
        self._store = store
        self._client = client
        self._chat_model = chat_model

    def ask(self, question: str) -> str:
        # 1. Считаем вектор вопроса
        [q_vector] = self._embedder.embed_texts([question])

        # 2. Извлекаем топ-3 релевантных документа из базы Политеха
        contexts = self._store.search_similar(q_vector, limit=3)
        joined_context = "\n---\n".join(contexts)

        # 3. Формируем запрос к нейросети
        prompt = f"""
        Ты — официальный AI-ассистент Московского Политеха.
        Отвечай строго на основе предоставленного контекста. Если информации в контексте нет, 
        прямо скажи: «К сожалению, в базе знаний Политеха нет информации по этому вопросу».

        КОНТЕКСТ ИЗ БАЗЫ ЗНАНИЙ:
        {joined_context}

        ВОПРОС СТУДЕНТА:
        {question}
        """

        response = self._client.models.generate_content(
            model=self._chat_model,
            contents=prompt,
        )
        return response.text or "Не удалось сгенерировать ответ."
```
