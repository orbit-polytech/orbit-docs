# Архитектура BLoC и управление состоянием

В мобильном приложении **Orbit Mobile** для управления состоянием и событиями интерфейса принят паттерн **BLoC (`flutter_bloc`)** в связке с возможностями **Dart 3.5+ (`sealed classes`, pattern matching)** и политиками конкурентности **`bloc_concurrency`**.

---

## 1. Принцип однонаправленного потока данных (UDF)

В отличие от разрозненных вызовов `notifyListeners()` в `ChangeNotifier`, BLoC реализует строгий однонаправленный поток:

```text
┌─────────────────────────────────────────────────────────────┐
│                      Пользователь (UI)                      │
└──────────────────────────────┬──────────────────────────────┘
                               │ 1. Порождает Событие (Event)
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                        ScheduleBloc                         │
│  ├── Применяет Event Transformer (restartable / droppable)   │
│  ├── Вызывает доменные UseCases                             │
│  └── Вычисляет и эмитит новое Состояние (State)             │
└──────────────────────────────┬──────────────────────────────┘
                               │ 2. Эмитит Состояние (State)
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                Виджеты (BlocBuilder / Selector)              │
│  └── Перерисовывают только изменившиеся элементы дерева      │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Модель неисключающих состояний (`ScheduleReady`)

В реальном приложении экран расписания не может быть просто «либо загрузка, либо ошибка, либо данные». У студента одновременно могут быть:
1. Уже загруженное из локальной БД расписание;
2. Фоновый процесс проверки свежих данных с сервера;
3. Ошибка последней попытки обновления (например, пропал интернет в метро).

Если при ошибке обновления заменить весь экран на `ErrorScreen`, студент потеряет доступ к расписанию. Поэтому состояние `ScheduleReady` хранит данные и сопутствующие флаги:

```dart
// lib/features/schedule/presentation/bloc/schedule_state.dart
import 'package:flutter/foundation.dart';
import '../../../../core/error/failures.dart';
import '../../domain/entities/schedule_data.dart';

@immutable
sealed class ScheduleState {
  const ScheduleState();
}

/// Начальное состояние до инициализации
final class ScheduleInitial extends ScheduleState {
  const ScheduleInitial();
}

/// Первичная загрузка, когда на устройстве еще нет сохраненных данных
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

/// Экран полной ошибки (показывается ТОЛЬКО если локального кэша нет и сеть недоступна)
final class ScheduleEmptyError extends ScheduleState {
  const ScheduleEmptyError({required this.failure});
  final Failure failure;
}
```

---

## 3. Политики конкурентности и защита от гонок (Race Conditions)

### Проблема «Быстрого переключения групп»
Когда пользователь быстро вводит или выбирает группы в поиске (например: `261-331` ➔ `261-332` ➔ `261-334`), параллельно отправляется несколько сетевых запросов. Если ответ для группы `261-331` вернется позже из-за сетевой задержки, он может затереть выбранную группу `261-334`.

### Решение: Трансформер `restartable()`
Для предотвращения гонок используется библиотека `bloc_concurrency`:

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
    
    // При смене группы отменяем старый фоновый запрос
    on<ScheduleGroupChanged>(
      _onGroupChanged,
      transformer: restartable(),
    );

    // Ручной Pull-to-Refresh: игнорируем спам-повторы пока идет обновление
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
    // 1. Пытаемся мгновенно взять данные из Hive для новой группы
    final cachedResult = await _getCachedSchedule(event.group);
    
    if (cachedResult is Ok<ScheduleData>) {
      emit(ScheduleReady(
        data: cachedResult.value,
        selectedDayIndex: 0,
        lastCheckedAt: cachedResult.value.cachedAt,
        isRefreshing: true, // Сигнализируем о фоновой проверке
      ));
    } else {
      emit(const ScheduleLoadingFirstTime());
    }

    // 2. Запрашиваем актуальное расписание у Gateway
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

## 4. Обработка ошибок в доменном слое (`Failure`)

Ошибки типизируются через закрытую иерархию `sealed class Failure`:

```dart
// lib/core/error/failures.dart
import 'package:flutter/foundation.dart';

@immutable
sealed class Failure {
  const Failure({required this.message});
  final String message;
}

final class ServerFailure extends Failure {
  const ServerFailure({required super.message, this.statusCode});
  final int? statusCode;
}

final class NetworkFailure extends Failure {
  const NetworkFailure({required super.message});
}

final class CacheFailure extends Failure {
  const CacheFailure({required super.message});
}

final class AuthFailure extends Failure {
  const AuthFailure({required super.message});
}
```

---

## 5. Централизованное логирование: `AppBlocObserver`

Для отладки переходов состояний используется `AppBlocObserver`. 

> **Критическое правило безопасности:**  
> В логи категорически запрещено выводить токены авторизации (`access_token`, `refresh_token`), заголовки `Authorization` и тело личных сообщений в чате ИИ.

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
    developer.log(
      'Error in ${bloc.runtimeType}: $error',
      name: 'BLoC',
      error: error,
      stackTrace: stackTrace,
    );
    super.onError(bloc, error, stackTrace);
  }
}
```
