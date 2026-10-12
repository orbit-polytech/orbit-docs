# 13. Упаковка и CI/CD (Dockerfile, GHCR, infra)

Финальный этап разработки микросервиса — автоматизированная сборка контейнера, проверка безопасности и публикация в реестр контейнеров **GitHub Container Registry (GHCR)**.

В этой главе мы разберём эталонный многоэтапный `Dockerfile`, пайплайн GitHub Actions и интеграцию готового образа со стеком `infra`.

---

## 1. Эталонный Multi-stage Dockerfile

Многоэтапная сборка разделяет окружение компиляции и финальный рантайм-образ, снижая размер контейнера с 1 ГБ до 25 МБ:

```dockerfile
# =============================================================================
# ЭТАП 1: Сборка статического бинарника (Build Stage)
# =============================================================================
FROM golang:1.24-alpine AS builder

# Установка корневых сертификатов для исходящих TLS-вызовов
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /src

# Кэширование Go-модулей: копируем зависимости отдельно от исходного кода
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Компиляция статического бинарника без CGO с удалением отладочной информации (-s -w)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=v1.0.0" \
    -o /bin/tasks-service ./cmd/tasks-service

# Загрузка бинарника проверки здоровья
ADD https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.37/grpc_health_probe-linux-amd64 /bin/grpc_health_probe
RUN chmod +x /bin/grpc_health_probe

# =============================================================================
# ЭТАП 2: Финальный безопасный образ (Runtime Stage)
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.source="https://github.com/orbit-polytech/tasks-service"
LABEL org.opencontainers.image.description="Orbit AI Tasks Microservice"

# Запуск от непривилегированного пользователя (UID 65532)
USER nonroot:nonroot

# Копирование собранных артефактов
COPY --from=builder /bin/tasks-service /bin/tasks-service
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

EXPOSE 50051

HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/bin/grpc_health_probe", "-addr=:50051"]

ENTRYPOINT ["/bin/tasks-service"]
```

---

## 2. Пайплайн публикации в GHCR (GitHub Actions)

В каждом сервисном репозитории настроен автоматический пайплайн сборки при пуше тега релиза (например, `v1.0.0`):

```yaml
# .github/workflows/docker-publish.yml
name: Build and Push Container

on:
  push:
    tags:
      - 'v*.*.*'
    branches:
      - main

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: orbit-polytech/tasks-service

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata for Docker
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=semver,pattern={{version}}
            type=sha,prefix=sha-
            type=raw,value=latest,enable=${{ github.ref == 'refs/heads/main' }}

      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
```

---

## 3. Интеграция с `infra` (Docker Compose)

После публикации образа в GHCR любой разработчик команды может запустить весь стек сервисов через центральный репозиторий **`infra`**:

```yaml
# infra/docker-compose.yml
version: '3.8'

services:
  tasks-service:
    image: ghcr.io/orbit-polytech/tasks-service:latest
    container_name: tasks-service
    restart: unless-stopped
    ports:
      - "50051:50051"
    environment:
      - GRPC_PORT=50051
      - APP_ENV=development
      - LOG_LEVEL=debug
      - POSTGRES_DSN=postgres://tasks_user:tasks_pass@postgres:5432/tasks_db?sslmode=disable
      - RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
    depends_on:
      postgres:
        condition: service_healthy
      rabbitmq:
        condition: service_healthy
```

---

## 4. Антипаттерны: как нельзя и как правильно

### ❌ Как нельзя:
```dockerfile
# ОШИБКА 1: Запуск от root пользователя
# При уязвимости злоумышленник получает root доступ к хостовой машине

# ОШИБКА 2: Копирование всего кода до go mod download
COPY . .
RUN go mod download # Кэш зависимостей сбрасывается при каждой правке README!

# ОШИБКА 3: Одноэтапный образ с компилятором
FROM golang:1.24 # Образ размером 1.2 ГБ попадает в продакшен
```

### ✅ Как правильно:
```dockerfile
# Сначала кэширование зависимостей:
COPY go.mod go.sum ./
RUN go mod download

# Непривилегированный пользователь nonroot:
USER nonroot:nonroot

# Минималистичный distroless образ (25 МБ):
FROM gcr.io/distroless/static-debian12:nonroot
```

---

## Резюме

1. Dockerfile строится в два этапа (Multi-stage) на базе `gcr.io/distroless/static`.
2. Контейнер выполняется под пользователем `nonroot:nonroot` без прав суперпользователя.
3. CI/CD автоматически публикует образы в GHCR с семантическими тегами.
4. Развертывание координируется через `infra/docker-compose.yml`.
