# Архитектура MVVM и управление состоянием

В этом разделе детально рассматривается реализация паттерна **MVVM (Model-View-ViewModel)** и управление состоянием с помощью паттерна **Command** и sealed class **Result**.

---

## 1. Sealed class `Result<T>`

Любая операция репозитория или сервиса возвращает строгий тип `Result<T>`, который заставляет программиста явно обработать как успешный сценарий, так и ошибку:

```dart
// lib/utils/result.dart
sealed class Result<T> {
  const Result();

  factory Result.ok(T value) = Ok._;
  factory Result.error(Exception error) = Error._;
}

final class Ok<T> extends Result<T> {
  const Ok._(this.value);
  final T value;
}

final class Error<T> extends Result<T> {
  const Error._(this.error);
  final Exception error;
}
```

---

## 2. Паттерн Command для ViewModel

Паттерн `Command` оборачивает асинхронное действие во ViewModel, автоматически отслеживая флаги выполнения (`running`), результат (`result`) и ошибки. Это избавляет от ручного написания сотен строк с `isLoading = true; notifyListeners();`:

```dart
// lib/utils/command.dart
import 'package:flutter/foundation.dart';
import 'result.dart';

typedef CommandAction0<T> = Future<Result<T>> Function();

class Command0<T> extends ChangeNotifier {
  Command0(this._action);

  final CommandAction0<T> _action;
  bool _running = false;
  Result<T>? _result;

  bool get running => _running;
  Result<T>? get result => _result;

  Future<void> execute() async {
    if (_running) return;
    _running = true;
    _result = null;
    notifyListeners();

    try {
      _result = await _action();
    } catch (e) {
      _result = Result.error(Exception(e.toString()));
    } finally {
      _running = false;
      notifyListeners();
    }
  }
}
```

---

## 3. Пример ViewModel экрана задач

ViewModel зависит **только от репозитория**, хранит состояние экрана и предоставляет команды:

```dart
// lib/ui/tasks/view_models/tasks_viewmodel.dart
import 'package:flutter/foundation.dart';
import '../../../data/repositories/tasks/task_repository.dart';
import '../../../domain/models/task/task.dart';
import '../../../utils/command.dart';
import '../../../utils/result.dart';

class TasksViewModel extends ChangeNotifier {
  TasksViewModel({required TaskRepository taskRepository})
      : _taskRepository = taskRepository {
    loadTasks = Command0<List<Task>>(_loadTasks);
  }

  final TaskRepository _taskRepository;
  late final Command0<List<Task>> loadTasks;

  List<Task> _tasks = [];
  List<Task> get tasks => _tasks;

  Future<Result<List<Task>>> _loadTasks() async {
    final result = await _taskRepository.getTasks();
    if (result is Ok<List<Task>>) {
      _tasks = result.value;
    }
    return result;
  }
}
```

---

## 4. Виджет экрана (View)

View слушает изменения ViewModel через стандартный виджет `ListenableBuilder` и перерисовывает только нужную часть экрана:

```dart
// lib/ui/tasks/widgets/tasks_screen.dart
import 'package:flutter/material.dart';
import '../view_models/tasks_viewmodel.dart';
import '../../core/ui/error_indicator.dart';
import '../../../utils/result.dart';

class TasksScreen extends StatefulWidget {
  const TasksScreen({super.key, required this.viewModel});

  final TasksViewModel viewModel;

  @override
  State<TasksScreen> createState() => _TasksScreenState();
}

class _TasksScreenState extends State<TasksScreen> {
  @override
  void initState() {
    super.initState();
    widget.viewModel.loadTasks.execute();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Мои задачи')),
      body: ListenableBuilder(
        listenable: widget.viewModel.loadTasks,
        builder: (context, _) {
          final command = widget.viewModel.loadTasks;

          if (command.running) {
            return const Center(child: CircularProgressIndicator());
          }

          if (command.result is Error) {
            return ErrorIndicator(
              message: 'Не удалось загрузить задачи',
              onRetry: () => command.execute(),
            );
          }

          final tasks = widget.viewModel.tasks;
          if (tasks.isEmpty) {
            return const Center(child: Text('У вас пока нет задач'));
          }

          return ListView.builder(
            itemCount: tasks.length,
            itemBuilder: (context, index) {
              final task = tasks[index];
              return ListTile(
                title: Text(task.title),
                subtitle: Text('Дедлайн: ${task.dueAt.toLocal()}'),
              );
            },
          );
        },
      ),
    );
  }
}
```
