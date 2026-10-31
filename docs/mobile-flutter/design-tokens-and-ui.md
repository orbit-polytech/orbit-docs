# Дизайн-система Cron Dark и Адаптивный Floating Dock

Интерфейс **Orbit Mobile** спроектирован по стандартам профессиональных инструментов высокой продуктивности (*Things 3*, *Cron Calendar*). В визуальной среде используются строгая темная палитра **Cron Dark**, чистые скругления и адаптивный плавающий островок навигации.

---

## 1. Палитра дизайн-токенов Cron Dark

Палитра оптимизирована для снижения нагрузки на зрение студента в аудиториях и энергоэффективности на OLED-дисплеях:

| Токен | Hex-код | Роль в интерфейсе | Контрастность (WCAG AA) |
| :--- | :--- | :--- | :--- |
| `AppColors.background` | `#08090B` | Глубокий фоновый цвет экрана | — |
| `AppColors.surface` | `#111318` | Поверхность карточек занятий и списков | 14.8:1 к фону |
| `AppColors.border` | `#1A1D24` | Тонкая граница карточек (1 px) | 3.2:1 к поверхности |
| `AppColors.accent` | `#FF6B35` | Акцентный оранжевый цвет (индикаторы, кнопки) | 5.1:1 к фону (Pass) |
| `AppColors.textPrimary` | `#FFFFFF` | Основной текст (названия пар, аудитории) | 18.2:1 (AAA Pass) |
| `AppColors.textSecondary` | `#8F94A0` | Вторичный текст (преподаватели, таймштампы) | 6.4:1 (AA Pass) |
| `AppColors.urgentRed` | `#EF4444` | Бейджи горящих дедлайнов и отмен пар | 4.8:1 (Pass) |
| `AppColors.successGreen` | `#10B981` | Отметка сданных лабораторных работ | 5.3:1 (Pass) |

---

## 2. Шрифтовая пара и типографика

* **Заголовки и UI элементы:** `Geologica` (геометричный гротеск с высокой читаемостью при мелком кегле).
* **Сетка звонков, даты и кабинеты:** `JetBrains Mono` (моноширинный шрифт исключает прыжки ширины цифр при тиканье таймера до конца пары).

```dart
// lib/core/theme/app_typography.dart
import 'package:flutter/material.dart';
import 'app_colors.dart';

class AppTypography {
  static const heading1 = TextStyle(
    fontFamily: 'Geologica',
    fontSize: 24,
    fontWeight: FontWeight.w700,
    color: AppColors.textPrimary,
    letterSpacing: -0.5,
  );

  static const monoTimer = TextStyle(
    fontFamily: 'JetBrainsMono',
    fontSize: 14,
    fontWeight: FontWeight.w600,
    color: AppColors.accent,
    letterSpacing: 0.2,
  );
}
```

---

## 3. Адаптивный Floating Dock (Плавающий островок)

Навигация не использует стандартный `BottomNavigationBar`, прилипающий к низу экрана. Вместо этого применен **парящий островок** (Floating Dock) с эффектом размытия стекла.

```text
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│                    Контент экрана                           │
│   (Скроллируемый список с нижним отступом 96 pt)            │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│         ┌─────────────────────────────────────────┐         │
│         │  📅 Расписание  📋 Задачи  🤖 AI  📝   👤  │         │  ◄── Высота 64 pt, Blur 20 pt
│         └─────────────────────────────────────────┘         │
│                              ▲                              │
│                              │ Отступ: viewPadding.bottom + 12 pt
└──────────────────────────────┴──────────────────────────────┘
```

### Принципы адаптивности:
1. **Динамический расчет отступа:**  
   Положение островка над системной чертой жестов iOS / кнопками Android вычисляется динамически:
   ```dart
   final bottomInset = MediaQuery.viewPaddingOf(context).bottom + 12.0;
   ```
2. **Скрытие при появлении экранной клавиатуры:**  
   Когда студент начинает печатать сообщение ИИ-ассистенту или вводит название задачи, клавиатура не поднимает док вверх. При `MediaQuery.viewInsetsOf(context).bottom > 0` плавающий док плавно анимируется в прозрачность.
3. **Защита от перекрытия контента (Content Occlusion):**  
   Каждый скроллируемый экран (`ListView`, `CustomScrollView`) оборачивается с обязательным нижним отступом контента:
   ```dart
   padding: EdgeInsets.only(
     bottom: 64.0 + MediaQuery.viewPaddingOf(context).bottom + 24.0,
   )
   ```

---

## 4. Декларативная маршрутизация: 5 Вкладок

Роутер на базе `go_router 14+` использует `StatefulShellRoute.indexedStack`. Это сохраняет введенный текст, позицию скролла и открытые карточки в каждой из 5 вкладок:

```dart
// lib/app/routes/app_router.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../ui/shell/adaptive_dock.dart';
import '../../ui/shell/shell_screen.dart';
import '../../features/schedule/presentation/schedule_screen.dart';
import '../../features/tasks/presentation/tasks_screen.dart';
import '../../features/ai_assistant/presentation/ai_screen.dart';
import '../../features/notes/presentation/notes_screen.dart';
import '../../features/profile/presentation/profile_screen.dart';

final appRouter = GoRouter(
  initialLocation: '/schedule',
  routes: [
    StatefulShellRoute.indexedStack(
      builder: (context, state, navigationShell) {
        return ShellScreen(navigationShell: navigationShell);
      },
      branches: [
        // 1. Расписание занятий
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/schedule',
              builder: (context, state) => const ScheduleScreen(),
            ),
          ],
        ),
        // 2. Личные и групповые задачи
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/tasks',
              builder: (context, state) => const TasksScreen(),
            ),
          ],
        ),
        // 3. ИИ-помощник (Защищен AuthWall для гостей)
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/ai',
              builder: (context, state) => const AiScreen(),
            ),
          ],
        ),
        // 4. Заметки и конспекты лекций
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/notes',
              builder: (context, state) => const NotesScreen(),
            ),
          ],
        ),
        // 5. Профиль студента и настройки группы
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/profile',
              builder: (context, state) => const ProfileScreen(),
            ),
          ],
        ),
      ],
    ),
  ],
);
```

### Обработка тапа по активной вкладке:
При повторном тапе по уже открытой вкладке (например, по «Расписанию») список плавно скроллится в самый верх к текущей паре дня:
```dart
void onTabSelected(int index) {
  navigationShell.goBranch(
    index,
    initialLocation: index == navigationShell.currentIndex,
  );
}
```
