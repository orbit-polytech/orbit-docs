# 12. Healthcheck: grpc_health_probe, /healthz и distroless

В микросервисной архитектуре оркестратор (Docker Compose, Kubernetes) должен непрерывно знать реальное состояние каждого сервиса.

В этой главе мы разбираем настройку проверок жизнеспособности для **gRPC-микросервисов** (`grpc_health_probe`) и **API Gateway** (`/healthz`, `/readyz`), а также решение проблемы проверок в безопасных **distroless-образах**.

---

## 1. Концепция: Liveness vs Readiness

В современных системах проверки разделяются на два типа:

| Тип проверки | Назначение | Реакция оркестратора при сбое |
| :--- | :--- | :--- |
| **Liveness (Жив ли процесс)** | Проверяет, что процесс не завис в дедлоке и цикл событий работает. | Перезапуск контейнера (Restart). |
| **Readiness (Готов ли к трафику)** | Проверяет, что сервис прогрелся, подключился к БД и брокеру сообщений. | Временное исключение из балансировщика нагрузки. |

---

## 2. Healthcheck для gRPC-сервисов (`grpc_health_probe`)

Сервисы gRPC работают по бинарному протоколу HTTP/2 поверх TCP. Обычные утилиты `curl` или `wget` не умеют взаимодействовать с gRPC без поддержки протокола проверки здоровья.

### Стандартный протокол: `grpc.health.v1.Health`
Go-сервисы Orbit AI регистрируют стандартный сервер здоровья из пакета `google.golang.org/grpc/health`:

```go
// cmd/tasks-service/main.go
import (
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
)

// Регистрация health-сервера:
healthServer := health.NewServer()
grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

// Установка статуса после успешной инициализации всех адаптеров:
healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
healthServer.SetServingStatus("orbit.tasks.v1.TaskService", grpc_health_v1.HealthCheckResponse_SERVING)
```

> [!NOTE] Авторизация проверок здоровья
> Напоминаем: в главе 6.6 в `AuthInterceptor` мы специально добавили исключение для префикса `/grpc.health.v1.Health/`, чтобы оркестратор мог опрашивать здоровье сервиса без передачи пользовательского токена.

### Утилита `grpc_health_probe`
Для проверки статуса из командной строки и Docker используется утилита [grpc_health_probe](https://github.com/grpc-ecosystem/grpc-health-probe):

```bash
grpc_health_probe -addr=:50051 -service=orbit.tasks.v1.TaskService
# Вывод при успехе: status: SERVING (код возврата 0)
```

---

## 3. Healthcheck для API Gateway (`/healthz` и `/readyz`)

Для внешнего шлюза, работающего по протоколу HTTP, реализуются два стандартных эндпоинта:

```go
// gateway/internal/handler/health.go
package handler

import (
    "net/http"
    "github.com/redis/go-redis/v9"
)

type HealthHandler struct {
    redis *redis.Client
}

// Liveness probe: процесс запущен и отвечает
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte(`{"status":"alive"}`))
}

// Readiness probe: проверка критических внешних связей (Redis, Postgres)
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Проверка доступности кэша Redis:
    if err := h.redis.Ping(ctx).Err(); err != nil {
        http.Error(w, `{"status":"not ready","error":"redis unavailable"}`, http.StatusServiceUnavailable)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte(`{"status":"ready"}`))
}

// Регистрация в роутере Chi:
// r.Get("/healthz", healthHandler.Healthz) // стандарт Kubernetes / Docker
// r.Get("/health", healthHandler.Healthz)  // удобный алиас для разработчиков
// r.Get("/readyz", healthHandler.Readyz)
```

---

## 4. Ограничения Distroless-образов и решение

Для минимизации уязвимостей (CVE) все микросервисы Orbit AI собираются в ультра-защищённые образы **`gcr.io/distroless/static-debian12:nonroot`**.

### В чём заключается проблема:
В distroless-образах **отсутствуют**:
- Командные оболочки (`/bin/sh`, `/bin/bash`);
- Утилиты управления пакетами (`apt`, `apk`);
- Сетевые клиенты (`curl`, `wget`, `netcat`).

Если в Dockerfile написать стандартный `HEALTHCHECK CMD curl http://localhost:8080/healthz`, контейнер завершится аварийно с ошибкой `exec: "curl": executable file not found in $PATH`.

### Решение: Встраивание статических бинарников на этапе сборки

В Dockerfile используется многоэтапная сборка (Multi-stage build):
1. На этапе сборщика скачивается статический бинарник `grpc_health_probe`;
2. Бинарник копируется в финальный distroless-образ:

```dockerfile
# Этап 1: Сборка бинарника и подготовка утилит
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

# Сборка статического Go бинарника сервиса:
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/tasks-service ./cmd/tasks-service

# Скачивание статического бинарника grpc_health_probe:
ADD https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.37/grpc_health_probe-linux-amd64 /bin/grpc_health_probe
RUN chmod +x /bin/grpc_health_probe

# Этап 2: Финальный защищённый образ distroless
FROM gcr.io/distroless/static-debian12:nonroot

USER nonroot:nonroot

COPY --from=builder /bin/tasks-service /bin/tasks-service
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe

# Проверка здоровья выполняется через прямой запуск бинарника (без оболочки shell)
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/bin/grpc_health_probe", "-addr=:50051"]

ENTRYPOINT ["/bin/tasks-service"]
```

---

## 5. Антипаттерны: как нельзя и как правильно

### ❌ Как нельзя:
```dockerfile
# ОШИБКА 1: Использование тяжелого Ubuntu с curl ради healthcheck
FROM ubuntu:22.04 # Образ весит 500 МБ и содержит сотни уязвимостей CVE

# ОШИБКА 2: HEALTHCHECK через shell в distroless
HEALTHCHECK CMD /bin/sh -c "curl ..." # Упадёт, так как /bin/sh не существует!

# ОШИБКА 3: Всегда отдавать 200 OK из /readyz
# Оркестратор будет слать трафик на сервис, у которого отвалилась база данных
```

### ✅ Как правильно:
```dockerfile
# Минималистичный distroless образ (размер ~25 МБ):
FROM gcr.io/distroless/static-debian12:nonroot

# Прямой вызов скомпилированного бинарника в формате JSON массива:
HEALTHCHECK --interval=5s --timeout=3s \
    CMD ["/bin/grpc_health_probe", "-addr=:50051"]
```

---

## Резюме

1. gRPC-сервисы реализуют стандартный `grpc.health.v1.Health` и проверяются через `grpc_health_probe`.
2. Gateway реализует `/healthz` (liveness) и `/readyz` (readiness с проверкой Redis).
3. В distroless-образах бинарник проверки здоровья встраивается на этапе multi-stage сборки.
