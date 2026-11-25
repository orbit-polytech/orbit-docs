# Orbit AI — инженерная база знаний

Техническая документация, архитектурные стандарты и учебные материалы проекта **Orbit AI** (Московский Политех).

- Сайт: https://orbit-polytech.github.io/orbit-docs/
- Текущее состояние проекта: https://orbit-polytech.github.io/orbit-docs/project/current-state/
- [Роли и зоны ответственности](docs/project/roles.md)
- [Обзор и правила команды](docs/quickstart/overview.md)
- Быстрый старт: https://orbit-polytech.github.io/orbit-docs/quickstart/local-setup/
- Публичное API расписания: https://orbit-polytech.github.io/orbit-docs/gateway/schedule-api/

Документация разделяет работающий код в `main`, мобильную и веб-бету, компоненты в разработке и учебные примеры вроде `examples/tasks-service`. Статус каждого направления зафиксирован на странице текущего состояния.

## Локальная сборка

Рекомендуется использовать виртуальное окружение `.venv`:

```bash
python -m venv .venv
# Linux / macOS:
source .venv/bin/activate
# Windows PowerShell:
# .\.venv\Scripts\Activate.ps1

python -m pip install -r requirements.txt
mkdocs serve
```

Проверка перед Pull Request:

```bash
python scripts/check-doc-links.py
mkdocs build --strict
```

## Инфраструктура, AI и мобилка

- [Запуск стенда с AI](https://orbit-polytech.github.io/orbit-docs/how-to/ai-local-stack/)
- [Мониторинг и логи (Observability)](https://orbit-polytech.github.io/orbit-docs/contracts-infra/observability/)
- [AI API: SSE, диалоги и оценки](https://orbit-polytech.github.io/orbit-docs/gateway/ai-api/)
- [Ограничения реализации AI](https://orbit-polytech.github.io/orbit-docs/python-ai/current-limitations/)
- [Мобильное приложение Orbit](https://orbit-polytech.github.io/orbit-docs/mobile-flutter/overview/)

`task up` в `infra` поднимает базовый профиль `apps` (PostgreSQL, Redis, `jwt-keygen`, `postgres-init`, миграции `auth-migrate`, `users-migrate`, `schedule-migrate`, сервисы `auth-service`, `users-service`, `schedule-service`, `gateway-service` и Web/PWA-контейнер `web` на `:8088`); полный стенд с AI, мониторингом, `ops-bot` и туннелем запускается через `--profile all`.
