# 02. Слой Presentation и управление состоянием BLoC

В проекте **Orbit Mobile** управление состоянием клиентского приложения построено на библиотеке **`flutter_bloc`** и нативных возможностях **Dart 3.5+ (`sealed classes`, pattern matching)**.

Паттерн BLoC (Business Logic Component) обеспечивает однонаправленный поток данных (**Unidirectional Data Flow**), полностью изолируя верстку виджетов от бизнес-логики и сетевых запросов.

---

## 1. Поток данных в BLoC

```mermaid
flowchart LR
    UI["UI (Flutter Widget)"] -->|1. add(Event)| BLoC["ScheduleBloc"]
    BLoC -->|2. вызов UseCase| UseCase["Domain / UseCases"]
    UseCase -->|3. данные из Hive / Dio| BLoC
    BLoC -->|4. emit(State)| Stream["Stream состояний"]
    Stream -->|5. BlocBuilder (rebuild)| UI
```

---

## 2. Неисключающие состояния на примере `ScheduleBloc`

В реальном приложении экран расписания не может быть просто «либо загрузка, либо ошибка, либо данные». У студента одновременно могут быть:
1. Загруженное расписание из локальной базы Hive;
2. Фоновый процесс проверки свежих данных с сервера;
3. Ошибка последней попытки обновления (например, потеря связи в метро).

### 2.1 События (`schedule_event.dart`)
```dart
// lib/features/schedule/presentation/bloc/schedule_event.dart
import 'package:flutter/foundation.dart';

@immutable
sealed class ScheduleEvent {
  const ScheduleEvent();
}

/// Смена учебной группы (требует отмены предыдущих запросов через restartable)
final class ScheduleGroupChanged extends ScheduleEvent {
  const ScheduleGroupChanged({required this.group});
  final String group;
}

/// Ручной свайп-обновление (Pull-to-Refresh)
final class ScheduleRefreshTriggered extends ScheduleEvent {
  const ScheduleRefreshTriggered({required this.group});
  final String group;
}

/// Переключение дня недели в таймлайне (Пн, Вт, Ср...)
final class ScheduleDaySelected extends ScheduleEvent {
  const ScheduleDaySelected({required this.index});
  final int index;
}
```

### 2.2 Состояния (`schedule_state.dart`)
```dart
// lib/features/schedule/presentation/bloc/schedule_state.dart
import 'package:flutter/foundation.dart';
import '../../../../core/error/failures.dart';
import '../../domain/entities/schedule_data.dart';

@immutable
sealed class ScheduleState {
  const ScheduleState();
}

final class ScheduleInitial extends ScheduleState {
  const ScheduleInitial();
}

final class ScheduleLoadingFirstTime extends ScheduleState {
  const ScheduleLoadingFirstTime();
}

/// Основное состояние: данные есть, и параллельно может идти фоновое обновление
final class ScheduleReady extends ScheduleState {
  const ScheduleReady({
    required this.data,
    required this.selectedDayIndex,
    required this.lastCheckedAt,
    this.isRefreshing = false,
    this.refreshFailure,
  });

  final ScheduleData data;
  final int selectedDayIndex;
  final DateTime lastCheckedAt;
  final bool isRefreshing;
  final Failure? refreshFailure;

  ScheduleReady copyWith({
    ScheduleData? data,
    int? selectedDayIndex,
    DateTime? lastCheckedAt,
    bool? isRefreshing,
    Failure? refreshFailure,
  }) {
    return ScheduleReady(
      data: data ?? this.data,
      selectedDayIndex: selectedDayIndex ?? this.selectedDayIndex,
      lastCheckedAt: lastCheckedAt ?? this.lastCheckedAt,
      isRefreshing: isRefreshing ?? this.isRefreshing,
      refreshFailure: refreshFailure,
    );
  }
}

/// Полная ошибка (показывается только если локального кэша нет)
final class ScheduleEmptyError extends ScheduleState {
  const ScheduleEmptyError({required this.failure});
  final Failure failure;
}
```

### 2.3 Логика BLoC и защита от гонок (`bloc_concurrency`)
```dart
// lib/features/schedule/presentation/bloc/schedule_bloc.dart
import 'package:bloc_concurrency/bloc_concurrency.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'schedule_event.dart';
import 'schedule_state.dart';

class ScheduleBloc extends Bloc<ScheduleEvent, ScheduleState> {
  ScheduleBloc({
    required GetCachedScheduleUseCase getCachedSchedule,
    required SyncScheduleUseCase syncSchedule,
  })  : _getCachedSchedule = getCachedSchedule,
        _syncSchedule = syncSchedule,
        super(const ScheduleInitial()) {

    // При быстрой смене группы отменяем старый сетевой запрос
    on<ScheduleGroupChanged>(
      _onGroupChanged,
      transformer: restartable(),
    );

    // Pull-to-refresh: игнорируем спам повторных свайпов
    on<ScheduleRefreshTriggered>(
      _onRefreshTriggered,
      transformer: droppable(),
    );

    on<ScheduleDaySelected>(_onDaySelected);
  }

  final GetCachedScheduleUseCase _getCachedSchedule;
  final SyncScheduleUseCase _syncSchedule;

  Future<void> _onGroupChanged(
    ScheduleGroupChanged event,
    Emitter<ScheduleState> emit,
  ) async {
    final cachedResult = await _getCachedSchedule(event.group);

    if (cachedResult is Ok<ScheduleData>) {
      emit(ScheduleReady(
        data: cachedResult.value,
        selectedDayIndex: 0,
        lastCheckedAt: cachedResult.value.cachedAt,
        isRefreshing: true,
      ));
    } else {
      emit(const ScheduleLoadingFirstTime());
    }

    final syncResult = await _syncSchedule(event.group);

    if (state is ScheduleReady) {
      final current = state as ScheduleReady;
      emit(switch (syncResult) {
        Ok(:final value) => current.copyWith(
            data: value,
            lastCheckedAt: DateTime.now(),
            isRefreshing: false,
            refreshFailure: null,
          ),
        Error(:final error) => current.copyWith(
            isRefreshing: false,
            refreshFailure: error,
          ),
      });
    } else if (syncResult is Ok<ScheduleData>) {
      emit(ScheduleReady(
        data: syncResult.value,
        selectedDayIndex: 0,
        lastCheckedAt: DateTime.now(),
        isRefreshing: false,
      ));
    } else if (syncResult is Error<ScheduleData>) {
      emit(ScheduleEmptyError(failure: syncResult.error));
    }
  }

  void _onDaySelected(
    ScheduleDaySelected event,
    Emitter<ScheduleState> emit,
  ) {
    if (state is ScheduleReady) {
      final current = state as ScheduleReady;
      emit(current.copyWith(selectedDayIndex: event.index));
    }
  }

  Future<void> _onRefreshTriggered(
    ScheduleRefreshTriggered event,
    Emitter<ScheduleState> emit,
  ) async {
    if (state is! ScheduleReady) return;
    final current = state as ScheduleReady;

    emit(current.copyWith(isRefreshing: true));
    final syncResult = await _syncSchedule(event.group);

    emit(switch (syncResult) {
      Ok(:final value) => current.copyWith(
          data: value,
          lastCheckedAt: DateTime.now(),
          isRefreshing: false,
          refreshFailure: null,
        ),
      Error(:final error) => current.copyWith(
          isRefreshing: false,
          refreshFailure: error,
        ),
    });
  }
}
```

---

## 3. Интеграция с UI: `BlocConsumer` и Pattern Matching

```dart
// lib/features/schedule/presentation/screens/schedule_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../bloc/schedule_bloc.dart';
import '../bloc/schedule_state.dart';

class ScheduleScreen extends StatelessWidget {
  const ScheduleScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: BlocConsumer<ScheduleBloc, ScheduleState>(
        listener: (context, state) {
          // Показываем плашку только если фоновое обновление упало, а кэш уже на экране
          if (state is ScheduleReady && state.refreshFailure != null) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(
                content: Text('Не удалось обновить расписание. Показаны сохраненные данные.'),
                backgroundColor: Color(0xFF1E2128),
              ),
            );
          }
        },
        builder: (context, state) {
          return switch (state) {
            ScheduleInitial() || ScheduleLoadingFirstTime() =>
              const Center(child: CircularProgressIndicator()),
            ScheduleReady(:final data, :final isRefreshing) =>
              ScheduleContentView(
                data: data,
                isRefreshing: isRefreshing,
              ),
            ScheduleEmptyError(:final failure) =>
              ErrorEmptyView(message: failure.message),
          };
        },
      ),
    );
  }
}
```

---

## 4. Модульное тестирование BLoC (`bloc_test`)

Тесты должны проверять сохранение кэша и защиту от гонок:

```dart
// test/features/schedule/presentation/bloc/schedule_bloc_test.dart
import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

class MockGetCachedScheduleUseCase extends Mock implements GetCachedScheduleUseCase {}
class MockSyncScheduleUseCase extends Mock implements SyncScheduleUseCase {}

void main() {
  late MockGetCachedScheduleUseCase mockGetCached;
  late MockSyncScheduleUseCase mockSync;

  setUp(() {
    mockGetCached = MockGetCachedScheduleUseCase();
    mockSync = MockSyncScheduleUseCase();
  });

  blocTest<ScheduleBloc, ScheduleState>(
    'эмитит ScheduleReady с кэшем, затем обновляет свежими данными',
    build: () {
      when(() => mockGetCached('261-334')).thenAnswer((_) async => Ok(testCachedData));
      when(() => mockSync('261-334')).thenAnswer((_) async => Ok(testFreshData));
      return ScheduleBloc(
        getCachedSchedule: mockGetCached,
        syncSchedule: mockSync,
      );
    },
    act: (bloc) => bloc.add(const ScheduleGroupChanged(group: '261-334')),
    expect: () => [
      isA<ScheduleReady>()
          .having((s) => s.isRefreshing, 'isRefreshing', true)
          .having((s) => s.data, 'data', testCachedData),
      isA<ScheduleReady>()
          .having((s) => s.isRefreshing, 'isRefreshing', false)
          .having((s) => s.data, 'data', testFreshData),
    ],
  );

  blocTest<ScheduleBloc, ScheduleState>(
    'при ошибке фонового обновления сохраняет кэш на экране и проставляет refreshFailure',
    build: () {
      when(() => mockGetCached('261-334')).thenAnswer((_) async => Ok(testCachedData));
      when(() => mockSync('261-334')).thenAnswer((_) async => Error(NetworkFailure(message: 'No internet')));
      return ScheduleBloc(
        getCachedSchedule: mockGetCached,
        syncSchedule: mockSync,
      );
    },
    act: (bloc) => bloc.add(const ScheduleGroupChanged(group: '261-334')),
    expect: () => [
      isA<ScheduleReady>().having((s) => s.isRefreshing, 'isRefreshing', true),
      isA<ScheduleReady>()
          .having((s) => s.isRefreshing, 'isRefreshing', false)
          .having((s) => s.refreshFailure, 'refreshFailure', isNotNull),
    ],
  );
}
```

---

## Резюме для разработчика

1. BLoC управляет только локальными состояниями и событиями экрана — он **не заменяет** чистую архитектуру (Clean Architecture).
2. Используйте `ScheduleReady` с флагом `isRefreshing`, чтобы не стирать расписание с экрана при сбоях сети.
3. Всегда защищайте быстрые поисковые события с помощью `transformer: restartable()`.
4. Тестируйте поведение BLoC на гонки и сетевые сбои через `bloc_test`.
