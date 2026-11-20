# Orbit AI — инженерная база знаний

Техническая документация, архитектурные стандарты и учебные материалы проекта **Orbit AI** (Московский Политех).

- Сайт: https://orbit-polytech.github.io/orbit-docs/
- Текущее состояние проекта: https://orbit-polytech.github.io/orbit-docs/project/current-state/
- Быстрый старт: https://orbit-polytech.github.io/orbit-docs/quickstart/local-setup/
- Публичное API расписания: https://orbit-polytech.github.io/orbit-docs/gateway/schedule-api/

Документация разделяет работающий код в `main`, мобильную бету, компоненты в разработке и учебные примеры вроде `examples/tasks-service`. Статус каждого направления зафиксирован на странице текущего состояния.

## Локальная сборка

```bash
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

`task up` в `infra` поднимает базовый профиль `apps` (PostgreSQL, Redis, Auth, Schedule, Gateway); полный стенд с AI, мониторингом, `ops-bot` и туннелем запускается через `--profile all`.
