package taskgrpc

import (
	"context"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type actorContextKey struct{}

const HeaderUserID = "x-user-id"

// ContextWithActor помещает аутентифицированного пользователя (actor) в контекст.
func ContextWithActor(ctx context.Context, actor domain.OwnerID) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

// ActorFromContext извлекает пользователя из контекста запроса.
func ActorFromContext(ctx context.Context) (domain.OwnerID, error) {
	val, ok := ctx.Value(actorContextKey{}).(domain.OwnerID)
	if !ok || val.IsZero() {
		return domain.OwnerID{}, status.Error(codes.Unauthenticated, "субъект запроса (actor) отсутствует в контексте")
	}
	return val, nil
}

// --8<-- [start:auth_interceptor]
// AuthInterceptor извлекает заголовок x-user-id из метаданных шлюза Gateway и валидирует его.
//
// Граница доверия (Trust Boundary):
// tasks-service работает в закрытом внутреннем периметре сети (Service Mesh / VPC).
// Внешние запросы проходят аутентификацию по JWT на уровне API Gateway, который проверяет
// цифровую подпись, срок действия токена и прокидывает проверенный user_id в gRPC-заголовок x-user-id.
// tasks-service доверяет шлюзу, но выполняет строгую валидацию формата и длины идентификатора
// для защиты от инъекций, повреждённых заголовков и аномалий внутренней сети.
func AuthInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Пропускаем стандартный gRPC Health Checking Protocol без авторизации
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "отсутствуют gRPC-метаданные запроса")
		}

		userIDs := md.Get(HeaderUserID)
		if len(userIDs) == 0 {
			return nil, status.Error(codes.Unauthenticated, "заголовок x-user-id не передан в метаданных")
		}

		rawUserID := strings.TrimSpace(userIDs[0])
		if rawUserID == "" {
			return nil, status.Error(codes.Unauthenticated, "заголовок x-user-id не может быть пустым")
		}

		if len(rawUserID) > 64 {
			return nil, status.Error(codes.Unauthenticated, "длина x-user-id превышает максимально допустимые 64 символа")
		}

		if containsControlChars(rawUserID) {
			return nil, status.Error(codes.Unauthenticated, "заголовок x-user-id содержит недопустимые управляющие символы")
		}

		actor, err := domain.NewOwnerID(rawUserID)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "некорректный формат x-user-id")
		}

		return handler(ContextWithActor(ctx, actor), req)
	}
}

func containsControlChars(s string) bool {
	for _, r := range s {
		if r < 32 || r == 127 {
			return true
		}
	}
	return false
}

// --8<-- [end:auth_interceptor]

// --8<-- [start:logging_interceptor]
// LoggingInterceptor логирует каждый унарный gRPC-запрос со структурированными полями.
func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		code := status.Code(err)
		level := slog.LevelInfo
		if code != codes.OK {
			level = slog.LevelWarn
		}
		if code == codes.Internal {
			level = slog.LevelError
		}

		var reqID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("x-request-id"); len(vals) > 0 {
				reqID = vals[0]
			}
		}

		attrs := []slog.Attr{
			slog.String("method", info.FullMethod),
			slog.String("code", code.String()),
			slog.Duration("duration", duration),
		}
		if reqID != "" {
			attrs = append(attrs, slog.String("request_id", reqID))
		}

		logger.LogAttrs(ctx, level, "gRPC запрос", attrs...)

		return resp, err
	}
}

// --8<-- [end:logging_interceptor]

// --8<-- [start:recovery_interceptor]
// RecoveryInterceptor перехватывает паники в хендлерах, логирует стек и возвращает codes.Internal.
func RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorContext(ctx, "паника в gRPC хендлере",
					slog.Any("recover", r),
					slog.String("stack", string(debug.Stack())),
				)
				err = status.Error(codes.Internal, "внутренняя критическая ошибка сервиса")
			}
		}()
		return handler(ctx, req)
	}
}

// --8<-- [end:recovery_interceptor]
