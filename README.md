# Orbit AI — инженерная база знаний

Центральная техническая документация, архитектурные стандарты и учебные материалы проекта **Orbit AI** Московского Политеха.

- Сайт: https://orbit-polytech.github.io/orbit-docs/
- Текущее состояние проекта: https://orbit-polytech.github.io/orbit-docs/project/current-state/
- Быстрый старт: https://orbit-polytech.github.io/orbit-docs/quickstart/local-setup/

Документация описывает как уже реализованные компоненты, так и целевую архитектуру. Статус каждого направления фиксируется на странице текущего состояния.

## Локальная сборка

```bash
python -m pip install -r requirements.txt
mkdocs serve
```

Строгая проверка перед Pull Request:

```bash
python scripts/check-doc-links.py
mkdocs build --strict
```
