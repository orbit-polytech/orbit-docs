# Создание Go-микросервиса с нуля (Tutorial)

В этом практическом руководстве мы пошагово разберём создание нового микросервиса по стандарту Orbit AI: от инициализации модуля до запуска gRPC-сервера с Graceful Shutdown.

---

## Шаг 1. Инициализация модуля и структуры папок

Создадим проект и базовую структуру каталогов:

```bash
mkdir my-service && cd my-service
go mod init github.com/orbit-polytech/my-service

# Создаем структуру слоев
mkdir -p cmd/my-service
mkdir -p internal/config
mkdir -p internal/domain
mkdir -p internal/app
mkdir -p internal/adapters/postgres
mkdir -p internal/transport/grpc
mkdir -p migrations
```

Установим базовые зависимости:

```bash
go get google.golang.org/grpc@latest
go get google.golang.org/protobuf@latest
go get github.com/google/uuid@latest
```

---

## Шаг 2. Разработка чистого слоя Domain

В слое `internal/domain` мы создаем бизнес-сущности и Value Objects. Помним главное правило: **никаких внешних зависимостей, только стандартная библиотека Go**.

Создадим `internal/domain/entity.go`:

```go
package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyID      = errors.New("идентификатор не может быть пустым")
	ErrEmptyTitle   = errors.New("название не может быть пустым")
	ErrTitleTooLong = errors.New("название превышает 200 символов")
)

type ItemID struct {
	value string
}

func NewItemID(raw string) (ItemID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ItemID{}, ErrEmptyID
	}
	return ItemID{value: trimmed}, nil
}

func (id ItemID) String() string { return id.value }

type Item struct {
	id        ItemID
	title     string
	createdAt time.Time
}

func NewItem(id ItemID, title string, now time.Time) (*Item, error) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return nil, ErrEmptyTitle
	}
	if len(trimmed) > 200 {
		return nil, ErrTitleTooLong
	}
	return &Item{
		id:        id,
		title:     trimmed,
		createdAt: now.UTC(),
	}, nil
}

func (i *Item) ID() ItemID           { return i.id }
func (i *Item) Title() string        { return i.title }
func (i *Item) CreatedAt() time.Time { return i.createdAt }
```

---

## Шаг 3. Описание портов и сценариев в слое App

В каталоге `internal/app/ports.go` опишем интерфейсы, необходимые сценариям:

```go
package app

import (
	"context"
	"time"

	"github.com/orbit-polytech/my-service/internal/domain"
)

type ItemSaver interface {
	Save(ctx context.Context, item *domain.Item) error
}

type ItemFinder interface {
	FindByID(ctx context.Context, id domain.ItemID) (*domain.Item, error)
}

type Clock interface {
	Now() time.Time
}
```

Реализуем сценарий создания в `internal/app/create_item.go`:

```go
package app

import (
	"context"
	"fmt"

	"github.com/orbit-polytech/my-service/internal/domain"
)

type CreateItemUseCase struct {
	saver ItemSaver
	clock Clock
}

func NewCreateItemUseCase(saver ItemSaver, clock Clock) *CreateItemUseCase {
	return &CreateItemUseCase{saver: saver, clock: clock}
}

func (uc *CreateItemUseCase) Execute(ctx context.Context, rawID, title string) (*domain.Item, error) {
	id, err := domain.NewItemID(rawID)
	if err != nil {
		return nil, fmt.Errorf("ошибка валидации ID: %w", err)
	}

	item, err := domain.NewItem(id, title, uc.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сущности: %w", err)
	}

	if err := uc.saver.Save(ctx, item); err != nil {
		return nil, fmt.Errorf("сохранение в БД: %w", err)
	}

	return item, nil
}
```

---

## Шаг 4. Транспортный адаптер gRPC

В `internal/transport/grpc/handler.go` подключаем сгенерированный gRPC-сервер. Для соблюдения DIP объявляем интерфейс прямо на стороне потребителя:

```go
package grpc

import (
	"context"

	"github.com/orbit-polytech/my-service/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type itemCreator interface {
	Execute(ctx context.Context, rawID, title string) (*domain.Item, error)
}

type ItemHandler struct {
	creator itemCreator
}

func NewItemHandler(creator itemCreator) *ItemHandler {
	return &ItemHandler{creator: creator}
}
```

---

## Шаг 5. Точка сборки (Composition Root) и Graceful Shutdown

В `cmd/my-service/main.go` связываем все зависимости воедино и обеспечиваем безопасную остановку сервера без потери запросов при деплое:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	grpcServer := grpc.NewServer()

	// Обязательная регистрация health check для Docker и Kubernetes
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// gRPC reflection для отладки через grpcurl
	reflection.Register(grpcServer)

	addr := ":50051"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("не удалось занять порт", slog.String("error", err.Error()))
		os.Exit(1)
	}

	go func() {
		logger.Info("gRPC сервер слушает порт", slog.String("addr", addr))
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error("ошибка gRPC сервера", slog.String("error", err.Error()))
		}
	}()

	<-ctx.Done()
	logger.Info("получен сигнал остановки, выполняется Graceful Shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		logger.Info("сервер успешно остановлен")
	case <-shutdownCtx.Done():
		logger.Warn("таймаут graceful shutdown, принудительное завершение")
		grpcServer.Stop()
	}
}
```
