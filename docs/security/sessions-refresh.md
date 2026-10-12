# Сессии, ротация Refresh-токенов и Token Reuse Detection

В этом руководстве описывается механизм долгоживущих сессий, ротация Refresh-токенов и защита от их кражи (Token Reuse Detection).

---

## 1. Схема базы данных сессий (`auth-service`)

В PostgreSQL создаются две связанные таблицы:

```sql
CREATE SCHEMA IF NOT EXISTS auth;

-- Долгоживущая сессия конкретного устройства
CREATE TABLE auth.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    user_agent TEXT NOT NULL,
    ip_address INET NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ NULL DEFAULT NULL
);

-- Хранилище хэшей действующих и отозванных refresh-токенов
CREATE TABLE auth.refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES auth.sessions(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE, -- SHA-256 хэш токена
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL DEFAULT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_hash ON auth.refresh_tokens(token_hash);
```

---

## 2. Механизм скользящей ротации (Refresh Token Rotation)

Каждый раз, когда клиент отправляет Refresh-токен для получения нового Access-токена:
1. `auth-service` находит токен по `SHA256(raw_refresh_token)` в таблице `auth.refresh_tokens`.
2. Если токен валиден и не отозван:
   * Старый токен **немедленно помечается как отозванный (`revoked_at = NOW()`)**.
   * Генерируется **новый** Refresh-токен со свежим хэшем.
   * Выпускается новый 15-минутный JWT Access-токен.
   * Оба токена возвращаются клиенту.

---

## 3. Обнаружение повторного использования (Token Reuse Detection)

Что произойдет, если злоумышленник перехватил Refresh-токен студента?

**Сценарий атаки:**
1. Легитимный клиент использует Refresh-токен №1 $\rightarrow$ сервер выдает Refresh-токен №2, а токен №1 отзывает.
2. Злоумышленник пытается использовать украденный Refresh-токен №1.
3. Сервер видит, что пришел токен со статусом `revoked_at IS NOT NULL`.
4. **Срабатывает протокол безопасности**:
   * Сервер определяет, что произошла попытка повторного использования украденного токена.
   * Сервер **немедленно отзывает всю сессию (`sessions.revoked_at = NOW()`)** и аннулирует ВСЕ связанные токены этой сессии.
   * Злоумышленник блокируется, а легитимного студента приложение попросит войти заново через Telegram.

---

## 4. Чёрный список отозванных токенов в Redis (Blacklist)

Для немедленной блокировки скомпрометированного Access-токена до истечения его 15-минутного срока жизни (например, при нажатии «Выйти со всех устройств»), `gateway-service` сохраняет `jti` (JWT ID) отозванных токенов в Redis с TTL, равным оставшемуся времени жизни токена:

```bash
# Сохранение отозванного токена с автоматическим удалением через 15 минут
SET auth:blacklist:<jti> "revoked" EX 900
```

При проверке токена в Gateway выполняется быстрый O(1) запрос: если ключ `auth:blacklist:<jti>` существует в Redis, запрос немедленно отклоняется с ошибкой `401 Unauthorized`.
