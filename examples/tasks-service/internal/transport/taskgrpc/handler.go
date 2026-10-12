package taskgrpc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/usecase"
	tasksv1 "github.com/orbit-polytech/orbit-docs/examples/tasks-service/pkg/api/orbit/tasks/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// --8<-- [start:grpc_handler]
// TaskHandler реализует gRPC-сервер TaskServiceServer.
// Является тонким транспортным адаптером: извлекает actor из контекста, валидирует DTO, вызывает usecase.
type TaskHandler struct {
	tasksv1.UnimplementedTaskServiceServer
	createUC     *usecase.CreateTaskUseCase
	getUC        *usecase.GetTaskUseCase
	listUC       *usecase.ListTasksUseCase
	completeUC   *usecase.CompleteTaskUseCase
	rescheduleUC *usecase.RescheduleTaskUseCase
	logger       *slog.Logger
}

func NewTaskHandler(
	createUC *usecase.CreateTaskUseCase,
	getUC *usecase.GetTaskUseCase,
	listUC *usecase.ListTasksUseCase,
	completeUC *usecase.CompleteTaskUseCase,
	rescheduleUC *usecase.RescheduleTaskUseCase,
	logger *slog.Logger,
) *TaskHandler {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &TaskHandler{
		createUC:     createUC,
		getUC:        getUC,
		listUC:       listUC,
		completeUC:   completeUC,
		rescheduleUC: rescheduleUC,
		logger:       logger,
	}
}

// CreateTask обрабатывает входящий gRPC-запрос на создание задачи.
func (h *TaskHandler) CreateTask(ctx context.Context, req *tasksv1.CreateTaskRequest) (*tasksv1.CreateTaskResponse, error) {
	actor, err := ActorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	priority, err := protoToDomainPriority(req.GetPriority())
	if err != nil {
		return nil, err
	}

	if req.GetDeadline() == nil {
		return nil, status.Error(codes.InvalidArgument, "дедлайн задачи обязателен")
	}
	if err := req.GetDeadline().CheckValid(); err != nil {
		return nil, status.Error(codes.InvalidArgument, "некорректный формат дедлайна (timestamppb)")
	}

	deadlineUTC := req.GetDeadline().AsTime().UTC()

	input := usecase.CreateTaskInput{
		Title:          req.GetTitle(),
		Description:    req.GetDescription(),
		Deadline:       deadlineUTC,
		Priority:       priority,
		IdempotencyKey: req.GetIdempotencyKey(),
	}

	task, err := h.createUC.Execute(ctx, actor, input)
	if err != nil {
		return nil, h.mapError(ctx, err)
	}

	return &tasksv1.CreateTaskResponse{
		Task: domainToProtoTask(task),
	}, nil
}

// GetTask возвращает задачу по её ID с проверкой прав доступа.
func (h *TaskHandler) GetTask(ctx context.Context, req *tasksv1.GetTaskRequest) (*tasksv1.GetTaskResponse, error) {
	actor, err := ActorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	task, err := h.getUC.Execute(ctx, actor, req.GetId())
	if err != nil {
		return nil, h.mapError(ctx, err)
	}

	return &tasksv1.GetTaskResponse{
		Task: domainToProtoTask(task),
	}, nil
}

// ListTasks возвращает список задач владельца с пагинацией.
func (h *TaskHandler) ListTasks(ctx context.Context, req *tasksv1.ListTasksRequest) (*tasksv1.ListTasksResponse, error) {
	actor, err := ActorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	limit := int(req.GetLimit())
	if limit == 0 {
		limit = 20
	}

	tasks, total, err := h.listUC.Execute(ctx, actor, limit, int(req.GetOffset()))
	if err != nil {
		return nil, h.mapError(ctx, err)
	}

	protoTasks := make([]*tasksv1.Task, 0, len(tasks))
	for _, t := range tasks {
		protoTasks = append(protoTasks, domainToProtoTask(t))
	}

	var totalCount int32
	if total > math.MaxInt32 {
		totalCount = math.MaxInt32
	} else {
		totalCount = int32(total)
	}

	return &tasksv1.ListTasksResponse{
		Tasks:      protoTasks,
		TotalCount: totalCount,
	}, nil
}

// CompleteTask отмечает задачу завершённой.
func (h *TaskHandler) CompleteTask(ctx context.Context, req *tasksv1.CompleteTaskRequest) (*tasksv1.CompleteTaskResponse, error) {
	actor, err := ActorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	task, err := h.completeUC.Execute(ctx, actor, req.GetId())
	if err != nil {
		return nil, h.mapError(ctx, err)
	}

	return &tasksv1.CompleteTaskResponse{
		Task: domainToProtoTask(task),
	}, nil
}

// RescheduleTask переносит дедлайн задачи.
func (h *TaskHandler) RescheduleTask(ctx context.Context, req *tasksv1.RescheduleTaskRequest) (*tasksv1.RescheduleTaskResponse, error) {
	actor, err := ActorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetNewDeadline() == nil {
		return nil, status.Error(codes.InvalidArgument, "новый дедлайн задачи обязателен")
	}
	if err := req.GetNewDeadline().CheckValid(); err != nil {
		return nil, status.Error(codes.InvalidArgument, "некорректный формат нового дедлайна (timestamppb)")
	}

	newDeadlineUTC := req.GetNewDeadline().AsTime().UTC()

	task, err := h.rescheduleUC.Execute(ctx, actor, req.GetId(), newDeadlineUTC)
	if err != nil {
		return nil, h.mapError(ctx, err)
	}

	return &tasksv1.RescheduleTaskResponse{
		Task: domainToProtoTask(task),
	}, nil
}

// --8<-- [end:grpc_handler]

// --8<-- [start:grpc_error_mapping]
// mapError переводит доменные ошибки в точные коды gRPC, не допуская утечки внутреннего контекста.
func (h *TaskHandler) mapError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "запрос отменён клиентом")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "превышен таймаут выполнения запроса")
	case errors.Is(err, domain.ErrEmptyTaskID):
		return status.Error(codes.InvalidArgument, domain.ErrEmptyTaskID.Error())
	case errors.Is(err, domain.ErrEmptyOwnerID):
		return status.Error(codes.Unauthenticated, domain.ErrEmptyOwnerID.Error())
	case errors.Is(err, domain.ErrEmptyTitle):
		return status.Error(codes.InvalidArgument, domain.ErrEmptyTitle.Error())
	case errors.Is(err, domain.ErrTitleTooLong):
		return status.Error(codes.InvalidArgument, domain.ErrTitleTooLong.Error())
	case errors.Is(err, domain.ErrDescriptionTooLong):
		return status.Error(codes.InvalidArgument, domain.ErrDescriptionTooLong.Error())
	case errors.Is(err, domain.ErrInvalidDeadline):
		return status.Error(codes.InvalidArgument, domain.ErrInvalidDeadline.Error())
	case errors.Is(err, domain.ErrInvalidPriority):
		return status.Error(codes.InvalidArgument, domain.ErrInvalidPriority.Error())
	case errors.Is(err, domain.ErrInvalidPagination):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrTaskNotFound):
		return status.Error(codes.NotFound, domain.ErrTaskNotFound.Error())
	case errors.Is(err, domain.ErrTaskAlreadyCompleted):
		return status.Error(codes.FailedPrecondition, domain.ErrTaskAlreadyCompleted.Error())
	case errors.Is(err, domain.ErrVersionConflict):
		return status.Error(codes.Aborted, domain.ErrVersionConflict.Error())
	default:
		h.logger.ErrorContext(ctx, "непредвиденная внутренняя ошибка сервера", slog.String("error", err.Error()))
		return status.Error(codes.Internal, "внутренняя ошибка сервиса")
	}
}

// --8<-- [end:grpc_error_mapping]

// --8<-- [start:grpc_mappers]
func protoToDomainPriority(p tasksv1.TaskPriority) (domain.TaskPriority, error) {
	switch p {
	case tasksv1.TaskPriority_TASK_PRIORITY_LOW:
		return domain.TaskPriorityLow, nil
	case tasksv1.TaskPriority_TASK_PRIORITY_MEDIUM:
		return domain.TaskPriorityMedium, nil
	case tasksv1.TaskPriority_TASK_PRIORITY_HIGH:
		return domain.TaskPriorityHigh, nil
	case tasksv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED:
		return domain.TaskPriorityUnspecified, status.Error(codes.InvalidArgument, "приоритет задачи обязателен и не может быть UNSPECIFIED")
	default:
		return domain.TaskPriorityUnspecified, status.Error(codes.InvalidArgument, "неизвестное значение приоритета задачи")
	}
}

func domainToProtoTask(t *domain.Task) *tasksv1.Task {
	return &tasksv1.Task{
		Id:          t.ID().String(),
		OwnerId:     t.OwnerID().String(),
		Title:       t.Title().String(),
		Description: t.Description(),
		Deadline:    timestamppb.New(t.Deadline().Time().UTC()),
		Priority:    domainToProtoPriority(t.Priority()),
		Status:      domainToProtoStatus(t.Status()),
		Version:     t.Version(),
		CreatedAt:   timestamppb.New(t.CreatedAt().UTC()),
	}
}

func domainToProtoPriority(p domain.TaskPriority) tasksv1.TaskPriority {
	switch p {
	case domain.TaskPriorityLow:
		return tasksv1.TaskPriority_TASK_PRIORITY_LOW
	case domain.TaskPriorityHigh:
		return tasksv1.TaskPriority_TASK_PRIORITY_HIGH
	case domain.TaskPriorityMedium:
		return tasksv1.TaskPriority_TASK_PRIORITY_MEDIUM
	default:
		return tasksv1.TaskPriority_TASK_PRIORITY_UNSPECIFIED
	}
}

func domainToProtoStatus(s domain.TaskStatus) tasksv1.TaskStatus {
	switch s {
	case domain.TaskStatusCompleted:
		return tasksv1.TaskStatus_TASK_STATUS_COMPLETED
	case domain.TaskStatusPending:
		return tasksv1.TaskStatus_TASK_STATUS_PENDING
	default:
		return tasksv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

// --8<-- [end:grpc_mappers]
