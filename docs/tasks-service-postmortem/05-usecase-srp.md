# Урок 5: Декомпозиция сценариев и SRP файлов

## 1. Как было сделано изначально (Файл-свалка нескольких сценариев)

В слое `internal/usecase` был создан файл `get_task.go`, в который со временем поместили:
* `GetTaskUseCase`
* `ListTasksUseCase`
* `CompleteTaskUseCase`
* `RescheduleTaskUseCase`
* Вспомогательную функцию проверки прав `loadOwned`

---

## 2. В чём скрытый дефект

Нарушение **Single Responsibility Principle (SRP)** на уровне организации файлов:

1. **Конфликты слияния в Git (Merge Conflicts)**: когда в команде из 10 человек один студент делает задачу «добавить пагинацию в список», а второй «изменить перенос дедлайна», они оба редактируют один и тот же файл `get_task.go`. При мердже PR возникает болезненный конфликт.
2. **Неочевидная навигация**: новый разработчик ищет, где реализован `CompleteTask`, ожидает увидеть файл `complete_task.go`, но не находит его и путается.

---

## 3. Как правильно делать в реальном сервисе

**Золотое правило: Один сценарий использования = Один файл.**

Каталог `internal/app/` (или `internal/usecase/`) структурируется строго плоско по именам сценариев:

```text
internal/app/
├── create_task.go        # CreateTaskUseCase
├── create_task_test.go
├── get_task.go           # GetTaskUseCase
├── get_task_test.go
├── list_tasks.go         # ListTasksUseCase
├── list_tasks_test.go
├── complete_task.go      # CompleteTaskUseCase
├── complete_task_test.go
├── reschedule_task.go    # RescheduleTaskUseCase
├── reschedule_task_test.go
└── ports.go              # Общие порты (ISP интерфейсы)
```

**Результат:**
* Разработчики могут параллельно работать над разными сценариями, не создавая взаимных конфликтов в Git.
* Время ревью PR сокращается в разы: сразу видно, какой конкретно сценарий затронут изменениями.
