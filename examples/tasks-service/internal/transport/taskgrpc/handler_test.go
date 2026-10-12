package taskgrpc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/infrastructure/idgen"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/infrastructure/inmemory"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/transport/taskgrpc"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/usecase"
	tasksv1 "github.com/orbit-polytech/orbit-docs/examples/tasks-service/pkg/api/orbit/tasks/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type staticClock struct {
	t time.Time
}

func (s *staticClock) Now() time.Time { return s.t }

// --8<-- [start:grpc_handler_tests]
func setupHandler(now time.Time) (*taskgrpc.TaskHandler, *inmemory.TaskRepository) {
	clock := &staticClock{t: now}
	idGen := idgen.NewUUIDGenerator()
	repo := inmemory.NewTaskRepository()
	events := inmemory.NewEventPublisher(nil)
	idempStore := inmemory.NewIdempotencyStore()

	createUC := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, idGen, idGen, clock)
	getUC := usecase.NewGetTaskUseCase(repo)
	listUC := usecase.NewListTasksUseCase(repo)
	completeUC := usecase.NewCompleteTaskUseCase(repo, repo)
	rescheduleUC := usecase.NewRescheduleTaskUseCase(repo, repo, clock)

	handler := taskgrpc.NewTaskHandler(createUC, getUC, listUC, completeUC, rescheduleUC, nil)
	return handler, repo
}

func newActorContext(userID string) context.Context {
	actor, _ := domain.NewOwnerID(userID)
	return taskgrpc.ContextWithActor(context.Background(), actor)
}

func TestTaskHandler_CreateTask_Success(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)

	ctx := newActorContext("student-100")
	req := &tasksv1.CreateTaskRequest{
		Title:          "Изучить Clean Architecture",
		Description:    "Глава 6 в orbit-docs",
		Deadline:       timestamppb.New(now.Add(24 * time.Hour)),
		Priority:       tasksv1.TaskPriority_TASK_PRIORITY_HIGH,
		IdempotencyKey: "req-key-1",
	}

	resp, err := handler.CreateTask(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.GetTask().GetId())
	assert.Equal(t, "student-100", resp.GetTask().GetOwnerId())
	assert.Equal(t, "Изучить Clean Architecture", resp.GetTask().GetTitle())
	assert.Equal(t, tasksv1.TaskStatus_TASK_STATUS_PENDING, resp.GetTask().GetStatus())
	assert.Equal(t, tasksv1.TaskPriority_TASK_PRIORITY_HIGH, resp.GetTask().GetPriority())
	assert.Equal(t, int64(1), resp.GetTask().GetVersion())
}

func TestTaskHandler_CreateTask_MissingActor(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)

	req := &tasksv1.CreateTaskRequest{
		Title:    "Задача без актора",
		Deadline: timestamppb.New(now.Add(24 * time.Hour)),
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_LOW,
	}

	resp, err := handler.CreateTask(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestTaskHandler_CreateTask_UnspecifiedPriority(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)
	ctx := newActorContext("student-100")

	req := &tasksv1.CreateTaskRequest{
		Title:    "Задача",
		Deadline: timestamppb.New(now.Add(24 * time.Hour)),
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED,
	}

	resp, err := handler.CreateTask(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "UNSPECIFIED")
}

func TestTaskHandler_CreateTask_MissingDeadline(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)
	ctx := newActorContext("student-100")

	req := &tasksv1.CreateTaskRequest{
		Title:    "Задача",
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_LOW,
	}

	resp, err := handler.CreateTask(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "дедлайн")
}

func TestTaskHandler_GetTask_ForeignTaskReturnsNotFound(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)

	// Создаём задачу от лица user-A
	ctxA := newActorContext("user-A")
	createResp, err := handler.CreateTask(ctxA, &tasksv1.CreateTaskRequest{
		Title:    "Секретная задача",
		Deadline: timestamppb.New(now.Add(5 * time.Hour)),
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_MEDIUM,
	})
	require.NoError(t, err)
	taskID := createResp.GetTask().GetId()

	// Пытаемся получить задачу от лица user-B -> Должен быть codes.NotFound
	ctxB := newActorContext("user-B")
	getResp, err := handler.GetTask(ctxB, &tasksv1.GetTaskRequest{Id: taskID})
	require.Error(t, err)
	assert.Nil(t, getResp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())

	// А от лица user-A задача успешно получается
	getRespA, err := handler.GetTask(ctxA, &tasksv1.GetTaskRequest{Id: taskID})
	require.NoError(t, err)
	assert.Equal(t, taskID, getRespA.GetTask().GetId())
}

func TestTaskHandler_ListTasks_OnlyOwnedTasksAndPagination(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)

	ctxA := newActorContext("user-A")
	ctxB := newActorContext("user-B")

	// user-A создает 3 задачи
	for i := 1; i <= 3; i++ {
		_, err := handler.CreateTask(ctxA, &tasksv1.CreateTaskRequest{
			Title:    "Задача А",
			Deadline: timestamppb.New(now.Add(time.Duration(i) * time.Hour)),
			Priority: tasksv1.TaskPriority_TASK_PRIORITY_LOW,
		})
		require.NoError(t, err)
	}

	// user-B создает 1 задачу
	_, err := handler.CreateTask(ctxB, &tasksv1.CreateTaskRequest{
		Title:    "Задача Б",
		Deadline: timestamppb.New(now.Add(10 * time.Hour)),
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_HIGH,
	})
	require.NoError(t, err)

	// user-A запрашивает список с лимитом 2
	listResp, err := handler.ListTasks(ctxA, &tasksv1.ListTasksRequest{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, listResp.GetTasks(), 2)
	assert.Equal(t, int32(3), listResp.GetTotalCount())

	// user-B запрашивает список
	listRespB, err := handler.ListTasks(ctxB, &tasksv1.ListTasksRequest{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, listRespB.GetTasks(), 1)
	assert.Equal(t, int32(1), listRespB.GetTotalCount())
}

func TestTaskHandler_ListTasks_ZeroLimitReturnsInvalidArgument(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)
	ctx := newActorContext("student-100")

	_, err := handler.ListTasks(ctx, &tasksv1.ListTasksRequest{Limit: 0, Offset: 0})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestTaskHandler_CompleteTask_AccessControlAndState(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)

	ctxA := newActorContext("user-A")
	ctxB := newActorContext("user-B")

	// Создаём задачу от лица user-A
	createResp, err := handler.CreateTask(ctxA, &tasksv1.CreateTaskRequest{
		Title:    "Задача для закрытия",
		Deadline: timestamppb.New(now.Add(5 * time.Hour)),
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_LOW,
	})
	require.NoError(t, err)
	taskID := createResp.GetTask().GetId()

	// user-B пытается закрыть чужую задачу -> NotFound
	_, err = handler.CompleteTask(ctxB, &tasksv1.CompleteTaskRequest{Id: taskID})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())

	// user-A закрывает свою задачу -> Успешно
	completeResp, err := handler.CompleteTask(ctxA, &tasksv1.CompleteTaskRequest{Id: taskID})
	require.NoError(t, err)
	assert.Equal(t, tasksv1.TaskStatus_TASK_STATUS_COMPLETED, completeResp.GetTask().GetStatus())
	assert.Equal(t, int64(2), completeResp.GetTask().GetVersion())

	// Повторное завершение возвращает FailedPrecondition
	_, err = handler.CompleteTask(ctxA, &tasksv1.CompleteTaskRequest{Id: taskID})
	require.Error(t, err)
	st, ok = status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestTaskHandler_RescheduleTask_AccessControl(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)

	ctxA := newActorContext("user-A")
	ctxB := newActorContext("user-B")

	createResp, err := handler.CreateTask(ctxA, &tasksv1.CreateTaskRequest{
		Title:    "Задача для переноса",
		Deadline: timestamppb.New(now.Add(5 * time.Hour)),
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_LOW,
	})
	require.NoError(t, err)
	taskID := createResp.GetTask().GetId()

	// user-B пытается перенести дедлайн чужой задачи -> NotFound
	_, err = handler.RescheduleTask(ctxB, &tasksv1.RescheduleTaskRequest{
		Id:          taskID,
		NewDeadline: timestamppb.New(now.Add(10 * time.Hour)),
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())

	// user-A переносит дедлайн в будущее -> Успешно
	reschedResp, err := handler.RescheduleTask(ctxA, &tasksv1.RescheduleTaskRequest{
		Id:          taskID,
		NewDeadline: timestamppb.New(now.Add(10 * time.Hour)),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), reschedResp.GetTask().GetVersion())

	// user-A переносит дедлайн в прошлое -> InvalidArgument
	_, err = handler.RescheduleTask(ctxA, &tasksv1.RescheduleTaskRequest{
		Id:          taskID,
		NewDeadline: timestamppb.New(now.Add(-1 * time.Hour)),
	})
	require.Error(t, err)
	st, ok = status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestTaskHandler_CreateTask_InvalidTimestamp(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)
	ctx := newActorContext("student-100")

	req := &tasksv1.CreateTaskRequest{
		Title:    "Задача с невалидным временем",
		Deadline: &timestamppb.Timestamp{Nanos: -1},
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_LOW,
	}

	resp, err := handler.CreateTask(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "некорректный формат дедлайна")
}

func TestTaskHandler_RescheduleTask_InvalidTimestamp(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)
	ctx := newActorContext("student-100")

	req := &tasksv1.RescheduleTaskRequest{
		Id:          "some-task-id",
		NewDeadline: &timestamppb.Timestamp{Nanos: -1},
	}

	resp, err := handler.RescheduleTask(ctx, req)
	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "некорректный формат нового дедлайна")
}

func TestTaskHandler_RescheduleTask_CompletedTask(t *testing.T) {
	now := time.Now().UTC()
	handler, _ := setupHandler(now)
	ctx := newActorContext("student-100")

	createResp, err := handler.CreateTask(ctx, &tasksv1.CreateTaskRequest{
		Title:    "Задача для закрытия и последующего переноса",
		Deadline: timestamppb.New(now.Add(5 * time.Hour)),
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_LOW,
	})
	require.NoError(t, err)
	taskID := createResp.GetTask().GetId()

	_, err = handler.CompleteTask(ctx, &tasksv1.CompleteTaskRequest{Id: taskID})
	require.NoError(t, err)

	_, err = handler.RescheduleTask(ctx, &tasksv1.RescheduleTaskRequest{
		Id:          taskID,
		NewDeadline: timestamppb.New(now.Add(10 * time.Hour)),
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

type conflictMockSaver struct {
	usecase.TaskFinder
}

func (c *conflictMockSaver) Save(ctx context.Context, task *domain.Task) error {
	return domain.ErrVersionConflict
}

func TestTaskHandler_VersionConflictMapping(t *testing.T) {
	now := time.Now().UTC()
	clock := &staticClock{t: now}
	idGen := idgen.NewUUIDGenerator()
	repo := inmemory.NewTaskRepository()
	events := inmemory.NewEventPublisher(nil)
	idempStore := inmemory.NewIdempotencyStore()

	createUC := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, idGen, idGen, clock)
	getUC := usecase.NewGetTaskUseCase(repo)
	listUC := usecase.NewListTasksUseCase(repo)
	conflictSaver := &conflictMockSaver{TaskFinder: repo}
	completeUC := usecase.NewCompleteTaskUseCase(repo, conflictSaver)
	rescheduleUC := usecase.NewRescheduleTaskUseCase(repo, conflictSaver, clock)

	handler := taskgrpc.NewTaskHandler(createUC, getUC, listUC, completeUC, rescheduleUC, nil)

	ctx := newActorContext("student-100")
	createResp, err := handler.CreateTask(ctx, &tasksv1.CreateTaskRequest{
		Title:    "Задача с конфликтом версий",
		Deadline: timestamppb.New(now.Add(5 * time.Hour)),
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_LOW,
	})
	require.NoError(t, err)
	taskID := createResp.GetTask().GetId()

	_, err = handler.CompleteTask(ctx, &tasksv1.CompleteTaskRequest{Id: taskID})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Aborted, st.Code())
	assert.Contains(t, st.Message(), domain.ErrVersionConflict.Error())
}

type failMockSaver struct{}

func (f *failMockSaver) Save(ctx context.Context, task *domain.Task) error {
	return errors.New("критический сбой дисковой подсистемы")
}

func TestTaskHandler_InternalErrorLogged(t *testing.T) {
	now := time.Now().UTC()
	clock := &staticClock{t: now}
	idGen := idgen.NewUUIDGenerator()
	repo := inmemory.NewTaskRepository()
	events := inmemory.NewEventPublisher(nil)
	idempStore := inmemory.NewIdempotencyStore()

	createUC := usecase.NewCreateTaskUseCase(&failMockSaver{}, repo, idempStore, events, idGen, idGen, clock)
	getUC := usecase.NewGetTaskUseCase(repo)
	listUC := usecase.NewListTasksUseCase(repo)
	completeUC := usecase.NewCompleteTaskUseCase(repo, repo)
	rescheduleUC := usecase.NewRescheduleTaskUseCase(repo, repo, clock)

	handler := taskgrpc.NewTaskHandler(createUC, getUC, listUC, completeUC, rescheduleUC, nil)

	ctx := newActorContext("student-100")
	_, err := handler.CreateTask(ctx, &tasksv1.CreateTaskRequest{
		Title:    "Задача на сбойном диске",
		Deadline: timestamppb.New(now.Add(5 * time.Hour)),
		Priority: tasksv1.TaskPriority_TASK_PRIORITY_LOW,
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "внутренняя ошибка сервиса")
}

// --8<-- [end:grpc_handler_tests]
