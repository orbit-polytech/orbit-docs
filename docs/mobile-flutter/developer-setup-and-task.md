# Руководство разработчика и Задачи Этапа 1 (Хасан Ризоев)

> **Статус**: Готово к передаче в разработку  
> **Роль**: Мобильный разработчик (Flutter)  
> **Автор / Архитектор**: Данил (Lead Architect / Team Lead)  
> **Репозиторий**: [github.com/orbit-polytech/mobile](https://github.com/orbit-polytech/mobile)  
> **Целевой стек**: Flutter 3.47+, Dart 3.13+, Android SDK (API 34/36), Web/Chrome, BLoC 9.1+, Hive CE 2.22+, GoRouter 18.1+

---

## 1. Введение и контекст задачи

Базовый каркас мобильного приложения `mobile` успешно инициализирован, проверен на тестовом билде Android 14 (Pixel 8) и выложен в репозиторий `origin/main`. Твоя задача на **Этапе 1** — развернуть чистую модульную архитектуру (Clean Architecture), реализовать дизайн-токены **Cron Dark**, создать адаптивный 5-вкладочный **Floating Dock** и построить первый экран расписания со счетчиком замен и офлайн-кэшированием в **Hive CE**.

---

## 2. Настройка рабочего окружения (Setup Guide)

### 2.1 Необходимое ПО
1. **Flutter SDK**: версия `3.47.x` или новее (канал `stable`), поставляется с Dart `3.13.x+`.
   - Путь к `flutter/bin` должен быть добавлен в системную переменную `PATH`.
2. **Android Studio** (Koala / Ladybug / Meerkat или новее):
   - В **SDK Manager -> SDK Platforms**: установить **Android 14.0 ("UpsideDownCake", API 34)** или **Android 15 (API 35)**.
   - В **SDK Manager -> SDK Tools**:
     - `Android SDK Build-Tools` (34.x / 35.x)
     - `Android SDK Command-line Tools (latest)`
     - `Android Emulator`
     - `Android SDK Platform-Tools`
     - `NDK (Side by side)` версии `28.2.13676358` или новее
     - `CMake` версии `3.22.1`

> [!CAUTION]
> **Важное предупреждение для VS Code:**  
> Если VS Code предложит установить расширение **«Extension Pack for Java»** — **НЕ устанавливай его** (или удали). Во Flutter весь код пишется на Dart, а Java-плагин перегружает систему анализом Gradle-скриптов и замедляет работу.

### 2.2 Настройка дискового пространства и AVD
Если на диске `C:` мало места, перенеси папку виртуальных устройств (AVD) на свободный диск (например, `F:` или `D:`):
```powershell
# 1. Создать целевую директорию
mkdir "F:\Soft\Android\avd"

# 2. Создать NTFS Junction (симлинк)
cmd /c mklink /J "C:\Users\<Ваш_Пользователь>\.android\avd" "F:\Soft\Android\avd"

# 3. Задать переменную среды
setx ANDROID_AVD_HOME "F:\Soft\Android\avd"
```

### 2.3 Рекомендованная конфигурация эмулятора Android
При создании AVD в Device Manager:
- **Устройство**: Pixel 8 или Pixel 7.
- **Образ (System Image)**: **Android 14 (API 34) Google Play Intel x86_64 Atom System Image** (строго **БЕЗ** приписки `16 KB Page Size`, так как 16KB preview нестабилен на Windows).
- **RAM**: 4096 MB (`hw.ramSize = 4096`).
- **VM Heap**: 512 MB (`vm.heapSize = 512`).
- **Graphics (GPU)**: На видеокартах AMD Radeon в файле `config.ini` эмулятора выставить `hw.gpu.mode = auto` или запустить с ключом `-gpu angle_indirect` (DirectX 11) для плавного рендеринга.

---

## 3. Быстрая альтернатива для вёрстки: Тестирование в Chrome

Чтобы не тратить ресурсы ПК на эмулятор во время вёрстки экранов и тестирования BLoC-состояний:
```powershell
cd mobile
flutter run -d chrome
```
В открывшемся браузере нажми **F12 -> `Ctrl + Shift + M`** и выбери формат смартфона (например, *iPhone 14 Pro* или *Pixel 7*). Работает моментальный **Hot Reload** (клавиша `r` в терминале).

---

## 4. Архитектурный стандарт проекта

Проект строится строго по стандарту **Feature-First Clean Architecture**:

```text
lib/
├── core/
│   ├── network/            # Dio клиент, Interceptors, обработка ошибок
│   ├── storage/            # Hive CE инициализация, SecureStorage (JWT)
│   ├── theme/              # Цветовая палитра Cron Dark, типографика, AppTheme
│   └── router/             # GoRouter с StatefulShellRoute (Floating Dock)
├── features/
│   ├── schedule/
│   │   ├── data/           # RemoteDataSource, LocalDataSource (Hive Box), DTO
│   │   ├── domain/         # Entity, ScheduleRepository интерфейс
│   │   └── presentation/   # ScheduleBloc, экраны, карточки пар, счетчик замен
│   ├── auth/               # Авторизация по талону/логину
│   ├── profile/            # Настройки, выбор группы
│   └── exams/              # Экзамены и зачеты
└── main.dart               # Инициализация Hive, DI и запуск приложения
```

### 4.1 Правила BLoC и защита от гонок
- Для событий обновления расписания использовать трансформер `restartable()` из пакета `bloc_concurrency`, чтобы отменять устаревшие сетевые запросы при быстром переключении дней недели.
- Все состояния объявлять через `sealed class` в стиле Dart 3:
```dart
sealed class ScheduleState extends Equatable {
  const ScheduleState();
}

final class ScheduleInitial extends ScheduleState {
  @override
  List<Object?> get props => [];
}

final class ScheduleLoading extends ScheduleState {
  @override
  List<Object?> get props => [];
}

final class ScheduleLoaded extends ScheduleState {
  final ScheduleEntity schedule;
  final bool isFromCache;
  final int replacementCount;

  const ScheduleLoaded({
    required this.schedule,
    required this.isFromCache,
    required this.replacementCount,
  });

  @override
  List<Object?> get props => [schedule, isFromCache, replacementCount];
}
```

---

## 5. Техническое задание: Задачи Этапа 1 для Хасана

### Задача 1.1: Дизайн-система Cron Dark и токены
- Создать `lib/core/theme/colors.dart`:
  - `bgPrimary`: `#08090B` (глубокий тёмный фон)
  - `bgSecondary`: `#111318` (карточки, модальные окна)
  - `accent`: `#FF6B35` (фирменный тёплый оранжевый)
  - `textPrimary`: `#FFFFFF`
  - `textSecondary`: `#8E95A5`
  - `border`: `#1E222D`
- Создать `lib/core/theme/app_theme.dart` с настройкой `ThemeData.dark(useMaterial3: true)`.

### Задача 1.2: Адаптивный Floating Dock (5 вкладок)
- Реализовать плавающий нижний док через `GoRouter` и `StatefulShellRoute.indexedStack`.
- Вкладки:
  1. **Расписание** (`/schedule`) — иконка календаря.
  2. **Задачи / Дедлайны** (`/tasks`) — иконка чек-листа.
  3. **Сессия / Экзамены** (`/exams`) — иконка академической шапочки.
  4. **AI-Помощник** (`/assistant`) — иконка искры / интеллекта.
  5. **Профиль** (`/profile`) — иконка пользователя.
- Эффект стекла: `BackdropFilter` с блюром `sigmaX: 12, sigmaY: 12`, скругление `24.0`, плавающий отступ от низа экрана `16.0`.

### Задача 1.3: Офлайн-кэш Hive CE
- Зарегистрировать TypeAdapter для моделей `ScheduleDay` и `Lesson`.
- Реализовать стратегию **Cache-First**:
  1. При старте моментально отдавать кэш из Hive Box (`schedule_box`).
  2. В фоне слать запрос к `schedule-service` API.
  3. При получении новых данных обновлять Hive и эмитить `ScheduleLoaded(isFromCache: false)`.

### Задача 1.4: Экран расписания с чипом замен
- Реализовать горизонтальный селектор дней недели (Пн – Сб) с индикацией текущего дня.
- Добавить счётчик изменений/замен расписания: оранжевый бейдж (например, `"2 замены сегодня"`).
- Скелетоны (shimmer) при первоначальной загрузке при пустом кэше.

---

## 6. Проверка качества перед Pull Request

Перед созданием PR разработчик обязан выполнить в терминале:

```powershell
# 1. Проверка строгой статической типизации (должно быть 0 issues)
flutter analyze

# 2. Запуск автотестов
flutter test
```

> [!IMPORTANT]
> В проекте включены строгие правила Dart Analysis (`strict-casts`, `strict-inference`, `strict-raw-types`). Неявные приведения типов (`dynamic`) и нетипизированные коллекции запрещены.
