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

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/config"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/infrastructure/idgen"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/infrastructure/inmemory"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/transport/taskgrpc"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/usecase"
	tasksv1 "github.com/orbit-polytech/orbit-docs/examples/tasks-service/pkg/api/orbit/tasks/v1"
	"google.golang.org/grpc"
)

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// --8<-- [start:composition_root]
func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("критическая ошибка выполнения сервиса", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(parentCtx context.Context) error {
	// 1. Загрузка и валидация конфигурации окружения
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("загрузка конфигурации: %w", err)
	}

	// 2. Инициализация структурированного логирования slog в зависимости от окружения
	logger := buildLogger(cfg)
	slog.SetDefault(logger)

	logger.Info("запуск tasks-service",
		slog.Int("port", cfg.GRPCPort),
		slog.String("env", cfg.Env),
		slog.String("log_level", cfg.LogLevel),
	)

	// 3. Graceful Shutdown через signal.NotifyContext
	ctx, stop := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Создание инфраструктурных адаптеров
	repo := inmemory.NewTaskRepository()
	idempStore := inmemory.NewIdempotencyStore()
	events := inmemory.NewEventPublisher(logger)
	idGenerator := idgen.NewUUIDGenerator()
	clock := systemClock{}

	// 5. Инициализация сценариев использования (Clean Architecture UseCases)
	createUC := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, idGenerator, idGenerator, clock)
	getUC := usecase.NewGetTaskUseCase(repo)
	listUC := usecase.NewListTasksUseCase(repo)
	completeUC := usecase.NewCompleteTaskUseCase(repo, repo)
	rescheduleUC := usecase.NewRescheduleTaskUseCase(repo, repo, clock)

	// 6. Инициализация gRPC транспортного адаптера и перехватчиков (Interceptors)
	taskHandler := taskgrpc.NewTaskHandler(createUC, getUC, listUC, completeUC, rescheduleUC, logger)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			taskgrpc.LoggingInterceptor(logger),
			taskgrpc.RecoveryInterceptor(logger),
			taskgrpc.AuthInterceptor(),
		),
	)
	tasksv1.RegisterTaskServiceServer(grpcServer, taskHandler)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("не удалось занять порт %s: %w", addr, err)
	}

	// 7. Запуск gRPC сервера
	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("gRPC сервер успешно слушает порт", slog.String("addr", addr))
		if serveErr := grpcServer.Serve(listener); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			serverErrCh <- serveErr
		}
	}()

	// 8. Ожидание завершения или критической ошибки
	select {
	case <-ctx.Done():
		logger.Info("получен сигнал завершения, остановка gRPC сервера...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stoppedCh := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stoppedCh)
		}()

		select {
		case <-stoppedCh:
			logger.Info("tasks-service успешно остановлен")
			return nil
		case <-shutdownCtx.Done():
			logger.Warn("таймаут graceful shutdown, принудительная остановка сервера")
			grpcServer.Stop()
			return nil
		}

	case serveErr := <-serverErrCh:
		return fmt.Errorf("ошибка gRPC сервера: %w", serveErr)
	}
}

// --8<-- [start:build_logger]
func buildLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	switch cfg.LogLevel {
	case config.LogLevelDebug:
		level = slog.LevelDebug
	case config.LogLevelWarn:
		level = slog.LevelWarn
	case config.LogLevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	if cfg.Env == config.EnvProduction {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

// --8<-- [end:build_logger]

// --8<-- [end:composition_root]
