# Архитектура BLoC и управление состоянием

В мобильном приложении **Orbit Mobile** управление состоянием клиентского интерфейса построено на библиотеке **`flutter_bloc`** и возможностях **Dart 3.5+ (`sealed classes`, pattern matching)**.

Паттерн BLoC (Business Logic Component) обеспечивает строгий однонаправленный поток данных (**Unidirectional Data Flow**), изолируя верстку виджетов от бизнес-логики и сетевых запросов.

---

## 1. Поток данных в BLoC

```mermaid
flowchart LR
    UI["UI (Flutter Widget)"] -->|1. add(Event)| BLoC["ScheduleBloc"]
    BLoC -->|2. вызов UseCase| UseCase["Domain / UseCases"]
    UseCase -->|3. данные из Hive / Dio| BLoC
    BLoC -->|4. emit(State)| Stream["Stream состояний"]
    Stream -->|5. BlocBuilder / Consumer| UI
```

---

## 2. Модель неисключающих состояний (`ScheduleReady`)

В реальном приложении экран расписания не может быть просто «либо загрузка, либо ошибка, либо данные». У студента одновременно могут быть:
1. Загруженное расписание из локального кэша Hive;
2. Фоновый процесс проверки свежих данных с сервера (`isRefreshing = true`);
3. Ошибка последней попытки обновления (например, потеря связи в метро), при которой само расписание **обязано оставаться на экране** с поясняющим баннером.

### 2.1 Модель состояний (`schedule_state.dart`)

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

/// Основное рабочее состояние экрана
final class ScheduleReady extends ScheduleState {
  const ScheduleReady({
    required this.group,
    required this.data,
    required this.selectedDayIndex,
    required this.lastCheckedAt,
    this.isRefreshing = false,
    this.refreshFailure,
  });

  final String group;
  final ScheduleData data;
  final int selectedDayIndex;
  final DateTime lastCheckedAt;
  final bool isRefreshing;
  final Failure? refreshFailure;

  ScheduleReady copyWith({
    String? group,
    ScheduleData? data,
    int? selectedDayIndex,
    DateTime? lastCheckedAt,
    bool? isRefreshing,
    Failure? refreshFailure,
    bool clearRefreshFailure = false, // Явный флаг для сброса ошибки
  }) {
    return ScheduleReady(
      group: group ?? this.group,
      data: data ?? this.data,
      selectedDayIndex: selectedDayIndex ?? this.selectedDayIndex,
      lastCheckedAt: lastCheckedAt ?? this.lastCheckedAt,
      isRefreshing: isRefreshing ?? this.isRefreshing,
      refreshFailure: clearRefreshFailure ? null : (refreshFailure ?? this.refreshFailure),
    );
  }
}

/// Экран полной ошибки (показывается только если локального кэша нет совсем)
final class ScheduleEmptyError extends ScheduleState {
  const ScheduleEmptyError({required this.failure});
  final Failure failure;
}
```

---

## 3. Защита от гонок состояний (Race Conditions)

### Проблема двух обработчиков:
Если студент инициировал ручное обновление группы А (`_onRefreshTriggered`), а затем быстро переключился на группу Б (`_onGroupChanged`), два обработчика работают параллельно. Без явного контроля ответ группы А может прийти позже и перетереть экран группы Б.

### Комплексное решение:
1. **Сетевая отмена (`CancelToken`):** При смене группы активный HTTP-запрос отменяется на уровне сокета.
2. **Проверка поколения и актуальности группы:** Перед каждым вызовом `emit` состояние проверяется **после** завершения `await`. Если активная группа изменилась за время сетевого ответа — старый результат отбрасывается.
3. **Сохранение выбранного дня:** При переключении дня недели флаг `refreshFailure` сохраняется и не сбрасывается втихую.

```dart
// lib/features/schedule/presentation/bloc/schedule_bloc.dart
import 'package:bloc_concurrency/bloc_concurrency.dart';
import 'package:dio/dio.dart';
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

    on<ScheduleGroupChanged>(
      _onGroupChanged,
      transformer: restartable(),
    );

    on<ScheduleRefreshTriggered>(
      _onRefreshTriggered,
      transformer: droppable(),
    );

    on<ScheduleDaySelected>(_onDaySelected);
  }

  final GetCachedScheduleUseCase _getCachedSchedule;
  final SyncScheduleUseCase _syncSchedule;

  CancelToken? _cancelToken;
  String? _currentGroup;

  Future<void> _onGroupChanged(
    ScheduleGroupChanged event,
    Emitter<ScheduleState> emit,
  ) async {
    _currentGroup = event.group;
    
    // 1. Отменяем предыдущий выполняющийся сетевой запрос
    _cancelToken?.cancel('Группа изменена на ${event.group}');
    _cancelToken = CancelToken();

    // 2. Мгновенно берем локальный кэш новой группы из Hive
    final cachedResult = await _getCachedSchedule(event.group);

    // Проверяем, не сменилась ли группа за время чтения Hive
    if (_currentGroup != event.group) return;

    if (cachedResult is Ok<ScheduleData>) {
      emit(ScheduleReady(
        group: event.group,
        data: cachedResult.value,
        selectedDayIndex: 0,
        lastCheckedAt: cachedResult.value.deviceCachedAt,
        isRefreshing: true,
      ));
    } else {
      emit(const ScheduleLoadingFirstTime());
    }

    // 3. Запрашиваем свежие данные у Gateway
    final syncResult = await _syncSchedule(event.group, cancelToken: _cancelToken);

    // КРИТИЧЕСКАЯ ПРОВЕРКА ПОСЛЕ AWAIT: отбрасываем ответ, если группа уже сменилась
    if (_currentGroup != event.group) return;

    if (state is ScheduleReady) {
      final current = state as ScheduleReady;
      emit(switch (syncResult) {
        Ok(:final value) => current.copyWith(
            data: value,
            lastCheckedAt: DateTime.now(),
            isRefreshing: false,
            clearRefreshFailure: true,
          ),
        Error(:final error) => current.copyWith(
            isRefreshing: false,
            refreshFailure: error,
          ),
      });
    } else if (syncResult is Ok<ScheduleData>) {
      emit(ScheduleReady(
        group: event.group,
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
      // Смена дня не сбрасывает ошибку фонового обновления
      emit(current.copyWith(selectedDayIndex: event.index));
    }
  }

  Future<void> _onRefreshTriggered(
    ScheduleRefreshTriggered event,
    Emitter<ScheduleState> emit,
  ) async {
    if (state is! ScheduleReady) return;
    final currentBefore = state as ScheduleReady;

    if (currentBefore.group != event.group || _currentGroup != event.group) return;

    emit(currentBefore.copyWith(isRefreshing: true));

    final syncResult = await _syncSchedule(event.group, cancelToken: _cancelToken);

    // Проверка актуальности после завершения сетевого запроса
    if (state is! ScheduleReady) return;
    final currentAfter = state as ScheduleReady;
    if (currentAfter.group != event.group || _currentGroup != event.group) return;

    emit(switch (syncResult) {
      Ok(:final value) => currentAfter.copyWith(
          data: value,
          lastCheckedAt: DateTime.now(),
          isRefreshing: false,
          clearRefreshFailure: true,
        ),
      Error(:final error) => currentAfter.copyWith(
          isRefreshing: false,
          refreshFailure: error,
        ),
    });
  }

  @override
  Future<void> close() {
    _cancelToken?.cancel('BLoC закрыт');
    return super.close();
  }
}
```

---

## 4. Централизованный `AppBlocObserver` и безопасность логов

```dart
// lib/app/observer.dart
import 'dart:developer' as developer;
import 'package:flutter_bloc/flutter_bloc.dart';

class AppBlocObserver extends BlocObserver {
  @override
  void onTransition(Bloc bloc, Transition transition) {
    super.onTransition(bloc, transition);
    developer.log(
      '${bloc.runtimeType}: ${transition.event.runtimeType} -> ${transition.nextState.runtimeType}',
      name: 'BLoC',
    );
  }

  @override
  void onError(BlocBase bloc, Object error, StackTrace stackTrace) {
    // В логи пишется только тип ошибки и стэк, без приватных полей и токенов
    developer.log(
      'Error in ${bloc.runtimeType}: ${error.runtimeType}',
      name: 'BLoC',
      error: error,
      stackTrace: stackTrace,
    );
    super.onError(bloc, error, stackTrace);
  }
}
```
