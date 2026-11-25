# Архитектура BLoC и управление состоянием

В клиентском приложении **Orbit** (`mobile`) управление состоянием построено на библиотеке **`flutter_bloc` (`^9.1.1`)**, **`bloc_concurrency` (`^0.3.0`)** и возможностях **Dart 3 (`sealed classes`, pattern matching)**.

Паттерн BLoC обеспечивает однонаправленный поток данных (**Unidirectional Data Flow**), изолируя виджеты от бизнес-логики и сетевых запросов.

---

## 1. Поток данных в BLoC

```mermaid
flowchart LR
    UI["UI (Flutter Widget)"] -->|1. add(Event)| BLoC["ScheduleBloc"]
    BLoC -->|2. вызов UseCase| UseCase["LoadWeekScheduleUseCase"]
    UseCase -->|3. поток данных Hive / Dio| BLoC
    BLoC -->|4. emit(State)| Stream["Stream состояний"]
    Stream -->|5. BlocBuilder / BlocConsumer| UI
```

---

## 2. Состояния `ScheduleBloc` (`schedule_state.dart`)

Файл: `lib/features/schedule/presentation/bloc/schedule/schedule_state.dart`

На экране расписания отображаются не просто «загрузка или ошибка», а комбинированное состояние с учётом кэша Hive, статуса синхронизации сервера (`READY` / `PENDING_SYNC`) и признака устаревания (`isStale`):

```dart
// lib/features/schedule/presentation/bloc/schedule/schedule_state.dart
sealed class ScheduleState extends Equatable {
  const ScheduleState();

  @override
  List<Object?> get props => [];
}

final class ScheduleInitial extends ScheduleState {
  const ScheduleInitial();
}

/// Группа ещё не выбрана — UI предлагает выбрать учебную группу
final class ScheduleSelectGroupRequired extends ScheduleState {
  const ScheduleSelectGroupRequired();
}

/// Первичная загрузка недели (когда готового кэша для этой цели ещё нет)
final class ScheduleLoading extends ScheduleState {
  const ScheduleLoading({
    required this.groupTitle,
    required this.weekMonday,
    this.serverStatus,
  });

  final String groupTitle;
  final DateTime weekMonday;
  final String? serverStatus;
}

/// Сервер ещё синхронизирует расписание с источником (`PENDING_SYNC`)
final class SchedulePendingSync extends ScheduleState {
  const SchedulePendingSync({
    required this.groupTitle,
    required this.weekMonday,
    this.serverStatus = 'PENDING_SYNC',
  });

  final String groupTitle;
  final DateTime weekMonday;
  final String? serverStatus;
}

/// Основное рабочее состояние экрана (кэш + фоновая синхронизация / ошибка)
final class ScheduleLoaded extends ScheduleState {
  const ScheduleLoaded({
    required this.groupTitle,
    required this.weekMonday,
    required this.selectedDate,
    required this.days,
    required this.syncStatus,
    this.isStale = false,
    this.lastUpdatedAt,
    this.lastCheckedAt,
    this.cachedAt,
    this.serverStatus,
    this.errorMessage,
    this.publishedFrom,
    this.publishedTo,
  });

  final String groupTitle;
  final DateTime weekMonday;
  final DateTime selectedDate;
  final List<AcademicDayEntity> days;
  final ScheduleSyncStatus syncStatus;
  final bool isStale;
  final DateTime? lastUpdatedAt;
  final DateTime? lastCheckedAt;
  final DateTime? cachedAt;
  final String? serverStatus;
  final String? errorMessage;
  final DateTime? publishedFrom;
  final DateTime? publishedTo;
}

/// Критическая ошибка при отсутствии локального кэша
final class ScheduleFailure extends ScheduleState {
  const ScheduleFailure({
    required this.groupTitle,
    required this.weekMonday,
    required this.message,
    this.selectedDate,
  });

  final String groupTitle;
  final DateTime weekMonday;
  final String message;
  final DateTime? selectedDate;
}
```

Ключевая особенность `ScheduleLoaded`:
- Поле `syncStatus` (`ScheduleSyncStatus`) показывает, идёт ли сейчас фоновое обновление, завершено ли оно успешно или завершилось ошибкой.
- Если сеть недоступна, кэш остаётся на экране (`ScheduleLoaded`), а ошибка отображается через `errorMessage` без сброса расписания.
- При переключении дня внутри уже загруженной недели (`ScheduleDaySelected`) обновляется только `selectedDate`, без повторного запроса на сервер.

---

## 3. Защита от гонок состояний: `restartable()`, `_currentRequestId` и `CancellationToken`

### Проблема сценария «А ➔ Б ➔ А»
Если пользователь быстро переключает группы или недели, более ранний асинхронный ответ может прийти позже нового и перезаписать актуальное состояние.

### Как это решено в `ScheduleBloc` (`lib/features/schedule/presentation/bloc/schedule/schedule_bloc.dart`)
1. **Трансформер `restartable()`:** все события загрузки сгруппированы под базовым типом `ScheduleLoadEvent` (`ScheduleStarted`, `ScheduleGroupChanged`, `ScheduleGroupPreviewRequested`, `ScheduleWeekChanged`, `ScheduleDateSelected`, `ScheduleTodayRequested`, `ScheduleRefreshRequested`) и обрабатываются с `transformer: restartable()` из `bloc_concurrency`.
2. **Счётчик запросов (`_currentRequestId`):** при каждом вызове `_loadSchedule` создаётся локальный `final requestId = ++_currentRequestId;`, который проверяется в `onData` и `onError` внутри `emit.forEach`.
3. **`CancellationToken` (`lib/core/cancellation/cancellation_token.dart`):** при новом запросе предыдущий токен отменяется (`_cancelToken?.cancel()`), а новый передаётся в `_loadWeekScheduleUseCase.execute(...)`. Это прерывает активный сетевой запрос и закрывает поток обновлений.

```dart
// lib/core/cancellation/cancellation_token.dart
class CancellationToken {
  bool _isCancelled = false;
  bool get isCancelled => _isCancelled;

  final Completer<void> _completer = Completer<void>();
  Future<void> get whenCancel => _completer.future;

  final List<void Function()> _listeners = [];

  void cancel() {
    if (_isCancelled) return;
    _isCancelled = true;
    if (!_completer.isCompleted) {
      _completer.complete();
    }
    for (final listener in List.of(_listeners)) {
      listener();
    }
    _listeners.clear();
  }

  void onCancel(void Function() listener) {
    if (_isCancelled) {
      listener();
    } else {
      _listeners.add(listener);
    }
  }
}
```

Фрагмент из `lib/features/schedule/presentation/bloc/schedule/schedule_bloc.dart`:

```dart
class ScheduleBloc extends Bloc<ScheduleEvent, ScheduleState> {
  ScheduleBloc({
    required this.repository,
    LoadWeekScheduleUseCase? loadWeekScheduleUseCase,
    DateTime Function()? nowProvider,
  }) : _loadWeekScheduleUseCase =
           loadWeekScheduleUseCase ?? LoadWeekScheduleUseCase(repository),
       _nowProvider = nowProvider ?? DateTime.now,
       super(const ScheduleInitial()) {
    on<ScheduleLoadEvent>(_onLoadEvent, transformer: restartable());
    on<ScheduleDaySelected>(_onDaySelected);
  }

  int _currentRequestId = 0;
  CancellationToken? _cancelToken;

  Future<void> _loadSchedule({
    required String groupTitle,
    required DateTime weekMonday,
    required DateTime selectedDate,
    bool forceRefresh = false,
    Completer<void>? refreshCompleter,
    required Emitter<ScheduleState> emit,
  }) async {
    final requestId = ++_currentRequestId;
    _refreshTracker.setActive(refreshCompleter);
    _cancelToken?.cancel();
    final cancelToken = CancellationToken();
    _cancelToken = cancelToken;

    _emitInitialLoadState(
      groupTitle: groupTitle,
      weekMonday: weekMonday,
      emit: emit,
    );

    try {
      await emit.forEach<ScheduleResult>(
        _loadWeekScheduleUseCase.execute(
          groupTitle: groupTitle,
          weekMonday: weekMonday,
          forceRefresh: forceRefresh,
          cancelToken: cancelToken,
        ),
        onData: (result) {
          if (requestId != _currentRequestId ||
              emit.isDone ||
              result.groupTitle != groupTitle ||
              !MoscowTime.isSameDay(result.weekMonday, weekMonday)) {
            return state;
          }
          return ScheduleStateMapper.mapResultToState(
            result: result,
            currentState: state,
            groupTitle: groupTitle,
            weekMonday: weekMonday,
            fallbackSelectedDate: selectedDate,
          );
        },
        onError: (error, _) {
          if (requestId != _currentRequestId || emit.isDone) {
            return state;
          }
          if (ScheduleStateMapper.isCancelledError(error) ||
              _cancelToken?.isCancelled == true) {
            return state;
          }
          return ScheduleStateMapper.mapErrorToState(
            error: error,
            groupTitle: groupTitle,
            weekMonday: weekMonday,
            currentState: state,
            selectedDate: selectedDate,
          );
        },
      );
    } finally {
      _refreshTracker.completeSpecific(refreshCompleter);
    }
  }
}
```

---

## 4. Другие ключевые BLoC в приложении

Помимо `ScheduleBloc`, в архитектуре приложения работают ещё два крупных BLoC (а для простых экранов вроде профиля и групп используются `Cubit`-ы: `UserProfileCubit`, `ScheduleGroupsCubit`, `IdentitiesCubit`):

* **`SessionBloc`** (`lib/features/auth/presentation/session/bloc/session_bloc.dart`):
  * Управляет глобальным жизненным циклом сессии и онбординга (`SessionUnknown`, `SessionOnboardingRequired`, `SessionGuest`, `SessionAuthenticated`).
  * Обрабатывает события последовательно (`transformer: sequential()`) и подписан на `ISessionRepository.sessionStream`.
  * При смене состояния в `OrbitApp` (`lib/app/app.dart`) автоматически подгружает или очищает профиль в `UserProfileCubit` и возвращает навигатор к корневому экрану при входе в аккаунт.
* **`AssistantBloc`** (`lib/features/assistant/presentation/bloc/assistant_bloc.dart`):
  * Координирует работу ИИ-ассистента через специализированные обработчики: `AssistantStreamHandler` (SSE-поток, стадия поиска, отмена, таймаут ожидания, таймер обратного отсчёта при `429` — `AssistantCooldownTicked`), `AssistantHistoryHandler` (загрузка, поиск и удаление диалогов) и `AssistantFeedbackHandler` (отправка оценок и расширенного фидбэка).

---

## 5. Безопасность логов при работе с сетью и BLoC

> **Правило безопасности логирования:**  
> В логи нельзя передавать целиком `DioException` или сырые объекты ошибок с заголовками запросов, так как они могут содержать `Authorization: Bearer ...` и токены сессии. Для диагностики передаются только безопасные сообщения и коды ошибок без секретов.
