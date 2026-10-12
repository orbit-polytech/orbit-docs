# Общие правила команды и организация репозиториев

Это базовые правила того, как устроена работа над кодовой базой Orbit AI: где создавать репозитории, как их называть, как вести ветки, коммиты и проводить код-ревью.

---

## Организация на GitHub и репозитории

Все проекты живут в единой GitHub-организации [**orbit-polytech**](https://github.com/orbit-polytech).

У каждого микросервиса и вспомогательного модуля — **свой отдельный репозиторий** (Polyrepo). Имена репозиториев пишутся строго в `kebab-case`. Поскольку организация уже называется `orbit-polytech`, префикс `orbit-` в названиях репозиториев **не используется** (чтобы не дублировать контекст в URL вида `github.com/orbit-polytech/auth-service`):

| Репозиторий | Что внутри | Основной стек |
|---|---|---|
| [`contracts`](https://github.com/orbit-polytech/contracts) | Proto-файлы gRPC-контрактов и сгенерированный код для Go, Python, Dart | Buf CLI, Protobuf v3 |
| [`infra`](https://github.com/orbit-polytech/infra) | Docker Compose для разработки, схемы PostgreSQL, RabbitMQ, Jaeger | Docker Compose, PostgreSQL 16, Redis 7, RabbitMQ |
| [`gateway-service`](https://github.com/orbit-polytech/gateway-service) | Единый публичный HTTP API шлюз для мобилки и внешних клиентов | Go 1.25, HTTP/REST, gRPC-клиенты, JWT verifier |
| [`auth-service`](https://github.com/orbit-polytech/auth-service) | Вход через Telegram, сессии, выпуск и ротация Ed25519 JWT | Go 1.25, gRPC, PostgreSQL, Redis |
| [`users-service`](https://github.com/orbit-polytech/users-service) | Профили студентов, учебные группы, права и роли | Go 1.25, gRPC, PostgreSQL |
| [`schedule-service`](https://github.com/orbit-polytech/schedule-service) | Парсинг и отдача расписания занятий, кэширование пар | Go 1.25, gRPC, Redis, PostgreSQL |
| [`navigation-service`](https://github.com/orbit-polytech/navigation-service) | Корпуса Политеха (БС, АВ, ПР), аудитории, построение маршрутов | Go 1.25, gRPC, PostgreSQL |
| [`tasks-service`](https://github.com/orbit-polytech/tasks-service) | Учебные задачи, дедлайны лабораторных работ (эталонный сервис) | Go 1.25, gRPC, Clean Architecture, PostgreSQL |
| [`notification-service`](https://github.com/orbit-polytech/notification-service) | Очередь отправки push-уведомлений и Telegram-сообщений | Go 1.25, gRPC, RabbitMQ Consumer |
| [`ai-service`](https://github.com/orbit-polytech/ai-service) | RAG-поиск по базе знаний вуза, Gemini API, эмбеддинги `vector(768)` | Python 3.12, FastAPI, gRPC, pgvector, uv |
| [`bot-service`](https://github.com/orbit-polytech/bot-service) | Официальный Telegram-бот для студентов Политеха | Python 3.12, aiogram 3, long polling / webhook |
| [`mobile`](https://github.com/orbit-polytech/mobile) | Кроссплатформенное мобильное приложение для iOS и Android | Flutter 3.x, Dart, MVVM / BLoC |
| [`orbit-docs`](https://github.com/orbit-polytech/orbit-docs) | Центральная база знаний и инженерная документация команды | MkDocs Material, Markdown |

> [!NOTE]
> Пока отдельный репозиторий сервиса не создан, его код можно прототипировать локально или в `orbit-docs/examples/`, но переносить в отдельный репозиторий нужно сразу, как только над сервисом начинает работать второй человек.

---

## Обязательное содержимое каждого репозитория

В корне любого репозитория организации обязательно присутствуют:

1. **`README.md`** — краткое описание сервиса, перечень портов, инструкция локального запуска за 5 минут, ссылка на документацию.
2. **`.env.example`** — исчерпывающий список всех переменных окружения с безопасными дефолтными значениями для локальной разработки.
3. **`.gitignore`** — исключает `.env`, локальные бинарники, файлы покрытия (`coverage.out`), временные директории IDE (`.idea/`, `.vscode/`).
4. **`Makefile`** (или `Taskfile`) — стандартные команды автоматизации: `run`, `test`, `lint`.
5. **`.github/workflows/ci.yml`** — автоматический пайплайн сборки, линтеров и запуска юнит-тестов при каждом PR.

---

## Ветки и GitHub Flow

Мы работаем по упрощённой модели **GitHub Flow**.

* Ветка **`main`** всегда стабильна: она собирается без предупреждений и проходит 100% автотестов.
* Любая разработка ведётся в короткоживущих тематических ветках, созданных от `main`.

Формат наименования веток: `<тип>/<номер-задачи>-<кратко>`

```bash
feat/42-task-deadlines
fix/57-refresh-token-reuse
docs/12-local-setup
chore/80-bump-go-1-25
```

Ветка живёт не более 2–3 дней. Если задача объёмная, разбивайте её на несколько последовательных небольших Pull Request.

---

## Соглашение о коммитах (Conventional Commits)

Сообщения коммитов оформляются строго по спецификации [Conventional Commits](https://www.conventionalcommits.org/ru/v1.0.0/):

```
<тип>(<область>): <краткое описание в повелительном наклонении>
```

Допустимые типы:
* `feat` — новая пользовательская или доменная функциональность;
* `fix` — исправление бага;
* `docs` — изменение документации;
* `refactor` — изменение кода без изменения поведения и исправления багов;
* `test` — добавление или исправление тестов;
* `chore` — рутинные задачи, обновление зависимостей, правка Makefile;
* `ci` — изменения в GitHub Actions и скриптах сборки;
* `perf` — оптимизация производительности.

### Примеры:

```bash
# Правильно:
feat(tasks): add deadline reminder calculation
fix(auth): reject reused refresh token immediately
test(postgres): cover optimistic lock conflict in task repo

# Неправильно (запрещено):
fix
update
правки в коде
final version 2
```

> [!TIP]
> Сообщение пишется на английском языке, в нижнем регистре, без точки в конце. Если изменение ломает обратную совместимость, добавляется восклицательный знак `!` после типа и абзац `BREAKING CHANGE:` в описании коммита.

Один коммит решает одну задачу. Никогда не смешивайте форматирование всего проекта и бизнес-логику в одном коммите.

---

## Регламент Pull Request

1. **Название PR** совпадает с названием основного коммита: `feat(tasks): add deadline reminder calculation`.
2. **Описание PR**: что сделано, зачем это нужно, как проверить работу вручную, ссылка на задачу (`Closes #42`).
3. **Объём изменений**: не более **300–400 строк** изменений (diff), исключая автоматически сгенерированные файлы proto/mock. Большие PR плохо проверяются и накапливают скрытые баги.
4. **Статус CI**: все проверки в GitHub Actions должны быть зелёными ДО запроса ревью.
5. **Аппрувы**: для слияния требуется минимум **1 подтверждение (Approve)**. Для изменений в `contracts`, авторизации и миграциях базы данных обязателен аппрув тимлида (Данил) или ведущих разработчиков.
6. **Слияние**: выполняется исключительно методом **Squash and merge**, после чего временная ветка удаляется.

---

## Шаблон Pull Request

Файл `.github/pull_request_template.md` подключается во все репозитории:

```markdown
## Что сделано

## Зачем

## Как проверить руками
1. 
2. 

## Чек-лист автора
- [ ] Тесты добавлены или обновлены (`go test ./...` / `pytest`)
- [ ] `.env.example` обновлён, если добавлены новые переменные
- [ ] Миграции БД обратимы (есть парные up и down скрипты)
- [ ] В логах нет токенов, паролей, кодов подтверждения
```
