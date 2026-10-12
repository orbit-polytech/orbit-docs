# Вёрстка экранов и виджеты (Tutorial)

В этом практическом руководстве мы разберём принципы создания адаптивных и оптимизированных виджетов Flutter в проекте Orbit Mobile.

---

## 1. Базовые правила вёрстки

1. **`const` конструкторы везде**: всегда используйте ключевое слово `const` для статических виджетов (`const SizedBox(height: 16)`). Flutter кэширует константные поддеревья и полностью исключает их из фазы перерисовки.
2. **Декомпозиция на отдельные классы**: вместо создания локальных вспомогательных методов внутри класса (`Widget _buildHeader()`), создавайте отдельные легковесные классы `StatelessWidget`. Приватные методы в классе виджета заставляют Flutter перерисовывать весь родительский виджет целиком при любом изменении состояния.
3. **Единые дизайн-токены**: цвета, шрифты и отступы никогда не задаются случайными числами (`Padding(padding: EdgeInsets.all(17.3))`). Все значения берутся из темы `Theme.of(context)` или дизайн-токенов `Dimens`.

---

## 2. Дизайн-токены Политеха (`ui/core/themes`)

```dart
// lib/ui/core/themes/dimens.dart
abstract final class Dimens {
  static const double paddingXS = 4.0;
  static const double paddingS = 8.0;
  static const double paddingM = 16.0;
  static const double paddingL = 24.0;
  static const double paddingXL = 32.0;

  static const double radiusS = 8.0;
  static const double radiusM = 12.0;
  static const double radiusL = 16.0;
}
```

```dart
// lib/ui/core/themes/colors.dart
import 'package:flutter/material.dart';

abstract final class OrbitColors {
  // Фирменные цвета Московского Политеха
  static const Color primaryBlue = Color(0xFF0055A5);
  static const Color accentGreen = Color(0xFF28A745);
  static const Color warningOrange = Color(0xFFFFC107);
  static const Color dangerRed = Color(0xFFDC3545);
  static const Color surfaceLight = Color(0xFFF8F9FA);
  static const Color textDark = Color(0xFF212529);
}
```

---

## 3. Эталонный переиспользуемый виджет карточки

Создадим `lib/ui/tasks/widgets/task_tile.dart`:

```dart
import 'package:flutter/material.dart';
import '../../../domain/models/task/task.dart';
import '../../core/themes/colors.dart';
import '../../core/themes/dimens.dart';

class TaskTile extends StatelessWidget {
  const TaskTile({
    super.key,
    required this.task,
    required this.onTap,
  });

  final Task task;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isOverdue = task.dueAt.isBefore(DateTime.now());

    return Card(
      margin: const EdgeInsets.symmetric(
        horizontal: Dimens.paddingM,
        vertical: Dimens.paddingS,
      ),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(Dimens.radiusM),
      ),
      elevation: 1,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(Dimens.radiusM),
        child: Padding(
          padding: const EdgeInsets.all(Dimens.paddingM),
          child: Row(
            children: [
              Icon(
                task.isCompleted ? Icons.check_circle : Icons.radio_button_unchecked,
                color: task.isCompleted ? OrbitColors.accentGreen : OrbitColors.primaryBlue,
                size: 28,
              ),
              const SizedBox(width: Dimens.paddingM),
              Expanded(
                child: Column(
                  crossContent: CrossAxisAlignment.start,
                  children: [
                    Text(
                      task.title,
                      style: theme.textTheme.titleMedium?.copyWith(
                        decoration: task.isCompleted ? TextDecoration.lineThrough : null,
                      ),
                    ),
                    const SizedBox(height: Dimens.paddingXS),
                    Text(
                      'Дедлайн: ${task.dueAt.day}.${task.dueAt.month}.${task.dueAt.year}',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: isOverdue ? OrbitColors.dangerRed : theme.hintColor,
                        fontWeight: isOverdue ? FontWeight.bold : FontWeight.normal,
                      ),
                    ),
                  ],
                ),
              ),
              const Icon(Icons.chevron_right, color: Colors.grey),
            ],
          ),
        ),
      ),
    );
  }
}
```
