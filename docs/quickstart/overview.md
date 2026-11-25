# Общие правила команды и организация репозиториев

Это базовые правила того, как устроена работа над кодовой базой Orbit AI: где создавать репозитории, как их называть, как вести ветки, коммиты и проводить код-ревью.

---

## Организация на GitHub и репозитории

Все проекты живут в единой GitHub-организации [**orbit-polytech**](https://github.com/orbit-polytech).

У каждого микросервиса и вспомогательного модуля — **свой отдельный репозиторий** (Polyrepo). Имена репозиториев пишутся строго в `kebab-case`. Поскольку организация уже называется `orbit-polytech`, префикс `orbit-` в названиях репозиториев **не используется** (чтобы не дублировать контекст в URL вида `github.com/orbit-polytech/auth-service`).

Сейчас в организации на GitHub **12 репозиториев** (включая служебный `.github` и приватный `ops-bot`), ещё **4 сервиса** запланированы на следующие этапы:

| Репозиторий / компонент | Что внутри | Основной стек и статус |
|---|---|---|
| [`contracts`](https://github.com/orbit-polytech/contracts) | Proto-файлы `auth.v1`, `users.v1`, `schedule.v1`, `ai.v1` и сгенерированный Go/Python-код (`v0.5.12`) | Buf CLI, Protobuf v3 (**работает**) |
| [`infra`](https://github.com/orbit-polytech/infra) | Compose (`v0.6.0`): PostgreSQL, Redis, Auth, Users, Schedule, Gateway, Web/PWA, AI, мониторинг и `ops-bot` | Docker Compose, PostgreSQL 16, Redis 7 (**работает**) |
| [`gateway-service`](https://github.com/orbit-polytech/gateway-service) | Единый публичный HTTP API шлюз: Auth (вкл. `email/change`), Users, Schedule, AI SSE + rate limit, Support | Go 1.26, HTTP/REST, gRPC, JWT Ed25519 (`contracts@v0.5.12`, **работает**) |
| [`auth-service`](https://github.com/orbit-polytech/auth-service) | Email OTP, смена email, Google OAuth, identities, сессии и Ed25519 JWT | Go 1.26, gRPC `:50051`, PostgreSQL, Redis (`contracts@v0.5.11`, **работает**) |
| [`users-service`](https://github.com/orbit-polytech/users-service) | Профили пользователей (`display_name`, `username`, учебные данные) | Go 1.26 (`1.26.10`), gRPC `:50052`, PostgreSQL `orbit_users` (`contracts@v0.5.12`, **работает**) |
| [`schedule-service`](https://github.com/orbit-polytech/schedule-service) | Публичное регулярное расписание, Redis-кэш и распределённая синхронизация | Go 1.26, gRPC `:50054`, PostgreSQL, Redis Streams (`contracts@v0.5.0`, **работает**) |
| [`ai-service`](https://github.com/orbit-polytech/ai-service) | Мультипровайдерный LLM, FastEmbed (384d), гибридный RAG, диалоги и оценки | Python 3.13+, `uv`, `grpc.aio`, `pgvector` (`contracts@v0.5.10`, **работает**) |
| [`knowledge-pipeline`](https://github.com/orbit-polytech/knowledge-pipeline) | ETL-конвейер (`orbit-etl`) для сбора `mospolytech.ru`, PDF и `curated_manual/` в `manifest.json` | Python 3.13+, `uv`, `httpx`, `pymupdf` (**работает**) |
| `ops-bot` *(приватный)* | Операционный Telegram-бот: алерты Alertmanager, обращения `/api/v1/support/issues`, триаж AI и дайджест | Python 3.13+, `uv`, aiogram 3, aiohttp (**работает**) |
| [`mobile`](https://github.com/orbit-polytech/mobile) | Клиент Orbit (`v0.1.27-beta+28`): Android APK и Web/PWA (`Dockerfile.web`, `web/nginx.conf`, контейнер `web :8088`) | Flutter 3.x / Dart `^3.13.7`, Clean Architecture / BLoC (**работает, бета**) |
| [`orbit-docs`](https://github.com/orbit-polytech/orbit-docs) | Центральная база знаний, учебный `examples/tasks-service` и инженерная документация | MkDocs Material, Markdown (**работает**) |
| [`tasks-service`](../project/current-state.md) *(план / пример в `orbit-docs`)* | Личные и групповые задачи, дедлайны (репозиторий не создан; эталон в `orbit-docs/examples/tasks-service`) | Go 1.26, gRPC, Clean Architecture, PostgreSQL (**учебный пример / план**) |
| `navigation-service` *(план)* | Корпуса Политеха (БС, АВ, ПР), аудитории, построение маршрутов | Go 1.26, gRPC, PostgreSQL (**запланировано**) |
| `notification-service` *(план)* | Очередь отправки push-уведомлений и Telegram-сообщений | Go 1.26, gRPC, RabbitMQ Consumer (**запланировано**) |
| `bot-service` *(план)* | Пользовательский Telegram-бот для студентов Политеха через Gateway | Python 3.13+, aiogram 3 (**запланировано**) |

> [!NOTE]
> В организации также есть служебный репозиторий [`.github`](https://github.com/orbit-polytech/.github). Пока отдельный репозиторий нового сервиса не создан, его код можно прототипировать локально или в `orbit-docs/examples/`, но переносить в отдельный репозиторий нужно сразу, как только над сервисом начинает работать второй человек.

---

Версии инструментов для конкретного репозитория берите из `go.mod`, `pyproject.toml`, `pubspec.yaml` и `Dockerfile`, а не из общей таблицы. Статусы сервисов — на странице [текущего состояния](../project/current-state.md), а черновик ролевой модели и RACI-матрицы — на странице [«Роли и зоны ответственности»](../project/roles.md) (составы направлений уточняются).

## Обязательное содержимое каждого репозитория

В корне любого репозитория организации обязательно присутствуют:

1. **`README.md`** — краткое описание сервиса, перечень портов, инструкция локального запуска за 5 минут, ссылка на документацию.
2. **`.env.example`** — исчерпывающий список всех переменных окружения с безопасными дефолтными значениями для локальной разработки (если сервису нужна конфигурация).
3. **`.gitignore`** — исключает `.env`, локальные бинарники, файлы покрытия (`coverage.out`), `.venv/`, временные директории IDE (`.idea/`, `.vscode/`).
4. **Конфигурация сборки и проверок под стек репозитория**:
   - для Go и `infra` — **`Taskfile.yml`** или **`Makefile`** (`run`, `test`, `lint`);
   - для Python (`ai-service`, `knowledge-pipeline`, `ops-bot`) — **`pyproject.toml`** + **`uv.lock`** (`uv run ruff check .`, `uv run mypy src`, `uv run pytest`);
   - для Flutter (`mobile`) — **`pubspec.yaml`** + **`tool/`** (`flutter analyze --fatal-infos`, `flutter test`, `python tool/check_version.py`).
5. **Workflow CI в `.github/workflows/`** — `ci.yml` (в `orbit-docs` — `docs-ci.yml`, в `mobile` также `release.yml`) для автоматической сборки, линтеров и тестов при каждом PR.

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
5. **Аппрувы**: для слияния требуется минимум **1 подтверждение (Approve)**. Для изменений в `contracts`, авторизации и миграциях базы данных обязателен аппрув Tech Lead (см. заготовку матрицы ответственности и направлений в [«Роли и зоны ответственности»](../project/roles.md)).
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


## Актуальный Python/AI и Telegram

AI-сервис и ETL базы знаний (`knowledge-pipeline`) уже реализованы и подключены к Gateway: [запуск](../how-to/ai-local-stack.md), [API](../gateway/ai-api.md). По части Telegram сейчас в `main` работает внутренний операционный бот для команды и мониторинга [`ops-bot`](../reference/services/ops-bot.md) (профили `ops` / `all`), тогда как пользовательский бот для студентов `bot-service` остаётся в планах.
