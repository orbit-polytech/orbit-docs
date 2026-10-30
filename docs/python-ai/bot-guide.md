# Telegram-бот на aiogram 3 (How-to Guide)

!!! warning "Учебный / целевой вариант"
    Примеры ниже сохранены для обучения и не являются инструкцией к текущему стенду. Gemini, старые env и схема БД относятся к альтернативному варианту; Telegram-бот ещё не реализован. Актуальная реализация — [паспорт ai-service](../reference/services/ai.md).

Репозиторий [**`bot-service`**](https://github.com/orbit-polytech/bot-service) реализует интерфейс Telegram-бота для студентов Московского Политеха с использованием современного асинхронного фреймворка **aiogram 3.x**.

---

## 1. Архитектура бота и разделение на роутеры

Бот построен на модульной структуре: бизнес-сценарии разносятся по независимым роутерам (`Router`), а не сваливаются в единый гигантский файл:

```text
bot-service/
├── src/
│   └── orbit_bot/
│       ├── __init__.py
│       ├── __main__.py          # Инициализация Bot, Dispatcher, долгоживущей сессии HTTP
│       ├── config.py            # Токен, адрес gateway-service
│       ├── routers/
│       │   ├── start.py         # Команды /start, /help, привязка аккаунта
│       │   ├── schedule.py      # Расписание пар студента
│       │   ├── tasks.py         # Задачи и дедлайны
│       │   └── ask.py           # Вопросы к AI-ассистенту
│       └── gateway_client.py    # Асинхронный HTTP-клиент к gateway-service
```

---

## 2. Инициализация и долгоживущий `httpx.AsyncClient`

> [!WARNING]
> Создавать новый HTTP-клиент на каждое входящее сообщение пользователя — грубейшая ошибка, приводящая к исчерпанию сокетов операционной системы. `httpx.AsyncClient` создаётся **один раз** при старте приложения и передаётся хэндлерам через контекст зависимости:

```python
# src/orbit_bot/__main__.py
import asyncio
import httpx
from aiogram import Bot, Dispatcher
from orbit_bot.config import Settings
from orbit_bot.routers import start, schedule, tasks, ask

async def main() -> None:
    cfg = Settings()
    bot = Bot(token=cfg.telegram_bot_token.get_secret_value())
    dp = Dispatcher()

    # Единый пул HTTP-соединений к шлюзу gateway-service
    async with httpx.AsyncClient(base_url=cfg.gateway_url, timeout=10.0) as http_client:
        # Пробрасываем клиент во все хэндлеры через workflow_data
        dp["http_client"] = http_client

        dp.include_router(start.router)
        dp.include_router(schedule.router)
        dp.include_router(tasks.router)
        dp.include_router(ask.router)

        await dp.start_polling(bot)

if __name__ == "__main__":
    asyncio.run(main())
```

---

## 3. Безопасная привязка Telegram-аккаунта

Когда бот запрашивает у студента контакт для привязки номера телефона, **обязательно** проверяется, что контакт отправлен именно этим пользователем, а не переслан от другого человека:

```python
# src/orbit_bot/routers/start.py
from aiogram import Router, F
from aiogram.types import Message
import httpx

router = Router()

@router.message(F.contact)
async def handle_contact(message: Message, http_client: httpx.AsyncClient) -> None:
    contact = message.contact
    if not contact:
        return

    # Защита от атаки подмены контакта:
    if contact.user_id != message.from_user.id:
        await message.answer("⚠️ Вы можете привязать только свой собственный номер телефона.")
        return

    # Отправляем проверенный контакт на шлюз gateway-service
    resp = await http_client.post(
        "/v1/auth/telegram-phone",
        json={
            "phone_number": contact.phone_number,
            "telegram_id": message.from_user.id,
        },
    )
    if resp.status_code == 200:
        await message.answer("✅ Номер успешно подтверждён! Добро пожаловать в Orbit AI.")
    else:
        await message.answer("❌ Ошибка привязки номера к учётной записи студента.")
```
