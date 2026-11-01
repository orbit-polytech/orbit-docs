# Дизайн-система: светлая тема и навигация

**Статус:** принципы согласованы, визуальное оформление выбирается. Примеры маршрутизации ниже описывают архитектуру, а не уже реализованный интерфейс.

**Светлая тема — по умолчанию.** Тёмная палитра Cron Dark из прежней версии не является обязательной. Точные цвета, шрифты, скругления, градиент и вид нижней панели нужно выбрать по [направлениям и референсам](design-directions-and-references.md).

## 1. Цветовые роли и варианты палитры

Роли задаются централизованно и используются всеми экранами. Не закрепляем цвета в отдельных виджетах и не выдаём предварительную палитру за проверенную дизайн-систему.

| Роль | Назначение | Что сравниваем в светлых макетах |
|---|---|---|
| `surface` / фон приложения | Основная поверхность | Почти белая, тёплая нейтральная или светлая с мягким градиентом в шапке |
| `surfaceContainer` | Карточки и сгруппированные блоки | Белая или слегка тонированная непрозрачная поверхность |
| `onSurface` | Основной текст | Тёмный читаемый текст |
| `onSurfaceVariant` | Вторичные сведения | Контрастный вторичный текст, не почти невидимый серый |
| `primary` / `onPrimary` | Основное действие и текст на нём | Оранжевый — кандидат; пара цветов проверяется вместе |
| `outline` | Значимые границы | Достаточно различимые поля и элементы управления |
| `error` / `onError` | Ошибки | Отдельная смысловая пара, не тот же цвет для всего |

Для обычного текста ориентир контраста — 4.5:1, для крупного — 3:1 по WCAG. На градиенте проверяется худший участок под текстом. Контраст для неподтверждённых цветов не объявляется измеренным или соответствующим AA/AAA. Цвет не является единственным способом обозначения статуса.

Основа: [Material — Color roles](https://m3.material.io/styles/color/roles), [Apple — Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility), [WCAG — Contrast (Minimum)](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html).

## 2. Типографика и размеры

Для первого сравнения достаточно системного шрифта. Geologica и JetBrains Mono остаются возможными кандидатами, а не обязательной парой. Пользовательские шрифты выбираются после проверки кириллицы, лицензии, читаемости и увеличенного текста. Для времени можно использовать табличные цифры без обязательного отдельного моноширинного шрифта.

Названия предметов не должны обрезаться из-за фиксированной высоты карточки. Поддерживаем системное масштабирование, семантические подписи и логичный порядок чтения. Для макетов ориентируемся на области нажатия 44×44 pt на iOS и 48×48 dp на Android; видимая иконка может быть меньше области нажатия.

Основа: [Apple — Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility), [Material — Accessible design](https://m3.material.io/foundations/accessible-design/overview).

## 3. Нижняя навигация: сравнить два варианта

Сравниваем стандартную нижнюю панель и Floating Dock на одинаковом экране. Ни стекло, ни blur, ни конкретная высота пока не обязательны. Пять направлений и их порядок остаются неизменными: Расписание, Задачи, AI, Заметки, Профиль. AI — вкладка, а не кнопка «+».

Material допускает 3–5 основных направлений в navigation bar: [Navigation bar](https://m3.material.io/components/navigation-bar/overview).

### Требования независимо от оформления

- Нижняя панель учитывает системные отступы и не закрывает последнюю карточку.
- Отступ контента рассчитывается по фактической высоте панели, её положению и safe area, а не всегда равен одной константе.
- При открытой клавиатуре заранее согласовывается поведение панели. Если она скрывается, невидимые элементы не должны продолжать принимать нажатия и фокус экранного диктора.
- Все вкладки имеют понятные подписи, активное состояние и семантику; центральный AI не затрудняет доступ к соседним разделам.
- При увеличении текста компоновка сохраняет читаемость, а не уменьшает подписи до неразличимого размера.
- Для больших окон можно выбрать другую компоновку навигации.
- Прозрачность, если выбрана, ограничивается слоем управления. Карточки и длинные ответы остаются читаемыми; есть непрозрачный вариант без эффекта.

Apple рекомендует отделять Liquid Glass навигации от слоя контента: [Materials](https://developer.apple.com/design/human-interface-guidelines/materials). Пользовательский Flutter blur не следует называть нативным Liquid Glass.

Детальные варианты, скриншоты и порядок выбора: [Дизайн: направления и референсы](design-directions-and-references.md).

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

### Повторный тап по активной вкладке:
Параметр `initialLocation: index == navigationShell.currentIndex` возвращает вложенную навигацию вкладки к корневому экрану ветки. Плавная прокрутка к текущей паре дня при этом реализуется непосредственно на самом экране расписания через привязанный `ScrollController.animateTo()`.
