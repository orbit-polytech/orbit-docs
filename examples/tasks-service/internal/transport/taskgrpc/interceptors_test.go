package taskgrpc_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/transport/taskgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor_Success(t *testing.T) {
	interceptor := taskgrpc.AuthInterceptor()

	md := metadata.Pairs(taskgrpc.HeaderUserID, "user-42")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		actor, err := taskgrpc.ActorFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, "user-42", actor.String())
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/orbit.tasks.v1.TaskService/CreateTask"}
	resp, err := interceptor(ctx, nil, info, handler)

	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, called)
}

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	interceptor := taskgrpc.AuthInterceptor()

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/orbit.tasks.v1.TaskService/CreateTask"}
	_, err := interceptor(context.Background(), nil, info, handler)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "отсутствуют gRPC-метаданные")
}

func TestAuthInterceptor_EmptyHeader(t *testing.T) {
	interceptor := taskgrpc.AuthInterceptor()

	md := metadata.Pairs(taskgrpc.HeaderUserID, "   ")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/orbit.tasks.v1.TaskService/CreateTask"}
	_, err := interceptor(ctx, nil, info, handler)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestRecoveryInterceptor_CatchesPanic(t *testing.T) {
	interceptor := taskgrpc.RecoveryInterceptor(slog.Default())

	handler := func(ctx context.Context, req any) (any, error) {
		panic("неожиданный сбой памяти")
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/orbit.tasks.v1.TaskService/CreateTask"}
	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "внутренняя критическая ошибка")
}

func TestAuthInterceptor_HealthCheckBypass(t *testing.T) {
	interceptor := taskgrpc.AuthInterceptor()

	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		return "health_ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}
	// Без всяких метаданных и x-user-id
	resp, err := interceptor(context.Background(), nil, info, handler)

	require.NoError(t, err)
	assert.Equal(t, "health_ok", resp)
	assert.True(t, called)
}

func TestAuthInterceptor_HeaderTooLong(t *testing.T) {
	interceptor := taskgrpc.AuthInterceptor()

	tooLongID := "user-" + string(make([]byte, 65))
	md := metadata.Pairs(taskgrpc.HeaderUserID, tooLongID)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/orbit.tasks.v1.TaskService/CreateTask"}
	_, err := interceptor(ctx, nil, info, handler)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "превышает максимально допустимые 64 символа")
}

func TestAuthInterceptor_ControlCharacters(t *testing.T) {
	interceptor := taskgrpc.AuthInterceptor()

	md := metadata.Pairs(taskgrpc.HeaderUserID, "user-42\nmalicious")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/orbit.tasks.v1.TaskService/CreateTask"}
	_, err := interceptor(ctx, nil, info, handler)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Contains(t, st.Message(), "управляющие символы")
}

func TestLoggingInterceptor(t *testing.T) {
	interceptor := taskgrpc.LoggingInterceptor(slog.Default())

	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		return "result", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/orbit.tasks.v1.TaskService/GetTask"}
	resp, err := interceptor(context.Background(), nil, info, handler)

	require.NoError(t, err)
	assert.Equal(t, "result", resp)
	assert.True(t, called)
}

func TestActorFromContext_Missing(t *testing.T) {
	actor, err := taskgrpc.ActorFromContext(context.Background())
	require.Error(t, err)
	assert.True(t, actor.IsZero())

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestActorFromContext_ZeroOwnerID(t *testing.T) {
	ctx := taskgrpc.ContextWithActor(context.Background(), domain.OwnerID{})
	actor, err := taskgrpc.ActorFromContext(ctx)
	require.Error(t, err)
	assert.True(t, actor.IsZero())
}
