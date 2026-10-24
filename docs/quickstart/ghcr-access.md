# Доступ к Docker-образам: авторизация в GHCR

Это руководство нужно перед первым запуском Orbit из **приватных** образов GitHub Container Registry (`ghcr.io`). Для публичных образов вход не требуется. Авторизация в GHCR не является входом в приложение Orbit и не заменяет настройку `.env`.

## Разработчику: получить доступ и войти

### 1. Проверьте доступ к пакетам

Попросите администратора предоставить вашей учётной записи или команде право **Read** на все шесть пакетов:

| Пакет | Назначение |
|---|---|
| `ghcr.io/orbit-polytech/auth-keygen` | Подготовка ключей Ed25519 |
| `ghcr.io/orbit-polytech/auth-migrate` | Миграции PostgreSQL |
| `ghcr.io/orbit-polytech/auth-service` | Сервис аутентификации |
| `ghcr.io/orbit-polytech/gateway-service` | HTTP API и Swagger |
| `ghcr.io/orbit-polytech/schedule-service` | Публичное расписание и синхронизация |
| `ghcr.io/orbit-polytech/schedule-migrate` | Миграции базы Schedule |

**Членство в организации само по себе не гарантирует доступ к каждому private-пакету.** PAT действует с правами владельца и не добавляет отсутствующие права.

### 2. Создайте персональный токен

1. Откройте GitHub → аватар → **Settings → Developer settings → Personal access tokens → Tokens (classic)**.
2. Выберите **Generate new token (classic)**. [Открыть создание токена с `read:packages`](https://github.com/settings/tokens/new?scopes=read:packages).
3. Укажите название, например `orbit-docker-pull`, и срок действия по политике команды, например 90 дней.
4. Выберите только **`read:packages`**. Для скачивания контейнерных образов не нужны `write:packages`, `delete:packages` или `repo`.
5. Создайте токен и сохраните его в менеджере паролей: GitHub показывает значение только при создании.

GitHub Packages использует для персонального входа PAT **classic**, не fine-grained PAT. Если организация требует SAML SSO, дополнительно разрешите токен для организации через **Configure SSO**. Если политика организации запрещает classic PAT, обратитесь к администратору, не пытайтесь обходить ограничение.

### 3. Авторизуйте Docker

В PowerShell, CMD, Bash или Zsh выполните, заменив `YOUR_GITHUB_USERNAME` своим логином GitHub:

```bash
docker login ghcr.io -u YOUR_GITHUB_USERNAME
```

При запросе `Password:` вставьте **созданный токен**, а не пароль от GitHub. Ввод может не отображаться — это нормально. Ожидаемый результат:

```text
Login Succeeded
```

!!! warning "Токен — секрет"
    Не добавляйте PAT в `.env`, Compose, Dockerfile, Git, скриншоты или переписку. Не передавайте его через `docker login -p ...`: значение окажется в аргументах команды и может остаться в истории. Токен нужен Docker на хосте, а не контейнерам Orbit.

Docker Desktop обычно использует системное хранилище учётных данных. На Docker Engine настройте credential helper: без него Docker может сохранить данные в `~/.docker/config.json` в Base64, **это не шифрование**. Не публикуйте этот файл. Используйте один и тот же пользовательский контекст для `docker login` и `task up`: `sudo`, WSL и другая настройка `DOCKER_CONFIG` могут использовать отдельное хранилище.

### 4. Проверьте скачивание всех образов

Для текущих стандартных тегов Compose:

```bash
docker pull ghcr.io/orbit-polytech/auth-keygen:latest
docker pull ghcr.io/orbit-polytech/auth-migrate:latest
docker pull ghcr.io/orbit-polytech/auth-service:latest
docker pull ghcr.io/orbit-polytech/gateway-service:latest
docker pull ghcr.io/orbit-polytech/schedule-service:latest
docker pull ghcr.io/orbit-polytech/schedule-migrate:latest
```

Ожидайте успешное скачивание либо сообщение `Image is up to date` для **каждого** образа. Если в локальном `.env` переопределены `JWT_KEYGEN_IMAGE`, `AUTH_MIGRATE_IMAGE`, `AUTH_SERVICE_IMAGE`, `GATEWAY_SERVICE_IMAGE`, `SCHEDULE_SERVICE_IMAGE` или `SCHEDULE_MIGRATE_IMAGE`, проверяйте соответствующие имена и теги. Авторизация едина для `ghcr.io`, а доступ проверяется отдельно для каждого пакета.

После [клонирования `infra` и заполнения `.env`](local-setup.md) запустите из каталога `infra`:

```bash
task up
task status
```

Продолжение: [проверка API и Swagger](local-setup.md).

### 5. Повторный вход, выход и отзыв

Авторизация обычно сохраняется между запусками. При истечении, отзыве или замене PAT повторите `docker login`. После изменения настроек SSO может потребоваться новая авторизация токена для организации.

Чтобы удалить локальные учётные данные реестра:

```bash
docker logout ghcr.io
```

`logout` не отзывает PAT на GitHub и не удаляет скачанные образы или работающие контейнеры. При утечке **отзовите PAT в GitHub**, создайте новый и повторите вход. Не делитесь одним токеном на всю команду: каждый использует свой.

## Администратору: предоставить доступ

Проверьте настройки **каждого из шести пакетов** на [странице Packages организации](https://github.com/orgs/orbit-polytech/packages).

Есть два способа выдать людям право чтения:

- **Наследование:** в **Package settings** включено **Inherit access from repository**. Право Read на связанный исходный репозиторий даёт доступ к пакету. Проверьте, что наследование действительно включено: одна ссылка на репозиторий не всегда означает наследование.
- **Отдельные права пакета:** если потребителям стенда не нужен доступ к исходникам, настройте доступ независимо от репозитория и добавьте команду организации через **Manage access → Invite teams or people**, роль **Read**. Если пакет наследует права, для независимой настройки потребуется отключить наследование, оценив влияние на существующие доступы.

Не выдавайте Write/Admin только ради скачивания образов. Убедитесь, что пользователь принял приглашение в организацию/команду и что политика организации допускает его PAT classic и срок действия.

!!! info "Доступ человека и GitHub Actions — разные настройки"
    **Manage Actions access → Add repository** предоставляет доступ workflow указанного репозитория через `GITHUB_TOKEN`. Добавление туда `infra` не предоставляет автоматически его участникам доступ с персональных компьютеров. В CI используйте `GITHUB_TOKEN` с необходимыми правами, а не персональный PAT разработчика.

Для проверки попросите участника с ролью Read выполнить вход и шесть `docker pull`. Успешное скачивание администратором не доказывает доступ остальных участников. Не меняйте видимость пакетов на Public только ради исправления ошибки доступа.

## Диагностика

| Симптом | Что проверить |
|---|---|
| `unauthorized`, `denied`, `403` | Вход именно в `ghcr.io`, логин владельца PAT, срок действия, `read:packages`, Read-доступ пользователя, SSO и политику организации |
| Вход успешен, pull запрещён | `Login Succeeded` не доказывает право чтения конкретного пакета; проверьте Manage access/Inherited access |
| Один образ скачивается, другой нет | Доступ отдельно к каждому из шести пакетов |
| `manifest unknown` / `not found` | Имя и тег, факт публикации; для private-пакета сначала исключите отсутствие доступа |
| `no matching manifest` | Архитектуру компьютера и платформы опубликованного образа; смена токена это не исправит |
| В одном терминале работает, в другом нет | Пользователя ОС, `sudo`, WSL и переменную `DOCKER_CONFIG` |
| `task up` сообщает об отсутствующей env-переменной | Заполните `.env` по Quickstart; это не ошибка авторизации GHCR |

При обращении за помощью укажите имя образа, тег и текст ошибки **без токена, содержимого Docker config и секретов `.env`**.

## Официальные источники

- [GitHub: аутентификация в Container registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [GitHub: доступ к пакетам и наследование прав](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility)
- [Docker: login и credential stores](https://docs.docker.com/reference/cli/docker/login/)
