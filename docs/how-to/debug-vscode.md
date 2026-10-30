# Как настроить отладку в VS Code (Debug)

Отладка через интерактивные точки останова (Breakpoints) в редакторе Visual Studio Code позволяет исследовать переменные в оперативной памяти, стек вызовов горутин и контекст запросов без засорения кодовой базы вызовами `fmt.Println` или `print()`.

---

## 1. Конфигурационный файл `.vscode/launch.json`

Создайте в корне вашего репозитория файл `.vscode/launch.json`. В зависимости от вашего трека используйте соответствующий блок конфигурации.

### Полная конфигурация для мультистека Orbit AI

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Go: Запуск tasks-service (Debug)",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/tasks-service/main.go",
      "envFile": "${workspaceFolder}/.env",
      "args": [],
      "showLog": true
    },
    {
      "name": "Go: Отладка текущего теста",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${fileDirname}",
      "envFile": "${workspaceFolder}/.env",
      "args": [
        "-test.v",
        "-test.run",
        "^${selectedText}$"
      ]
    },
    {
      "name": "Python: Запуск ai-service (gRPC)",
      "type": "debugpy",
      "request": "launch",
      "module": "orbit_ai",
      "cwd": "${workspaceFolder}",
      "args": [],
      "justMyCode": true,
      "envFile": "${workspaceFolder}/.env"
    },
    {
      "name": "Flutter: Мобильное приложение (Debug)",
      "type": "dart",
      "request": "launch",
      "program": "lib/main.dart",
      "flutterMode": "debug"
    }
  ],
  "compounds": [
    {
      "name": "Full-Stack: Go Backend + Flutter App",
      "configurations": [
        "Go: Запуск tasks-service (Debug)",
        "Flutter: Мобильное приложение (Debug)"
      ],
      "stopAll": true
    }
  ]
}
```

---

## 2. Пошаговый процесс отладки в Go (Delve)

### Шаг 1. Проверка установки отладчика Delve

Расширение Go для VS Code автоматически устанавливает отладчик `dlv`. При необходимости его можно установить вручную:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

---

### Шаг 2. Установка точки останова (Breakpoint)

1. Откройте интересующий вас файл, например `internal/transport/taskgrpc/handler.go`.
2. Кликните левой кнопкой мыши слева от номера строки (на строку `actorID, err := actorFromContext(ctx)`).
3. Появится красная точка — **Breakpoint**.

---

### Шаг 3. Запуск сессии отладки

1. Нажмите клавишу **F5** (или перейдите во вкладку `Run & Debug` через сочетание ++ctrl+shift+d++).
2. В верхнем выпадающем меню выберите **«Go: Запуск tasks-service (Debug)»**.
3. Сервер запустится и остановится в режиме ожидания запросов.

---

### Шаг 4. Отправка запроса и навигация

Отправьте gRPC-запрос через `grpcurl` в терминале:

```bash
grpcurl -plaintext -H "x-user-id: 018f2345-6789-7abc-def0-123456789abc" localhost:50051 orbit.tasks.v1.TaskService/ListTasks
```

Выполнение кода немедленно приостановится на отмеченной строке. В панели слева станут доступны:

- **Variables (Переменные):** текущие значения `ctx`, `req`, `h`.
- **Call Stack (Стек вызовов):** цепочка интерцепторов gRPC и вызовов функций.
- **Watch:** возможность написать любое выражение на Go (например, `req.GetLimit()`).

### Горячие клавиши управления:

| Клавиша | Действие | Описание |
| :--- | :--- | :--- |
| **F5** | Continue | Продолжить выполнение до следующей точки останова |
| **F10** | Step Over | Перейти на следующую строку без захода внутрь функции |
| **F11** | Step Into | Войти внутрь вызываемой функции |
| **Shift+F11** | Step Out | Завершить выполнение текущей функции и выйти на уровень выше |
| **Shift+F5** | Stop | Остановить сессию отладки |

---

## 3. Сравнение: Как нельзя и Как правильно

<div class="grid" markdown>

<div class="card" markdown>
### :x: Как нельзя

- **Вставлять десятки `fmt.Println("HERE 1")` по всему коду.** Логи загрязняют консоль, замедляют сервис и часто случайно попадают в коммиты и Pull Request.
- **Запускать отладчик без указания `envFile: "${workspaceFolder}/.env"`.** Сервер упадет с ошибкой невалидной конфигурации при старте `config.Load()`.
- **Отлаживать логику вслепую через пересборку Docker-образа каждый раз.** Это занимает 2–3 минуты на одну итерацию вместо 2 секунд по клавише F5.
</div>

<div class="card" markdown>
### :white_check_mark: Как правильно

- **Использовать нативный отладчик Delve (Go) и debugpy (Python)** с точками останова и панелью Watch.
- **Использовать конфигурацию `mode: "test"`** для пошаговой отладки сложных падающих юнит-тестов.
- **Настраивать составные запуски (compounds)** для одновременной отладки клиент-серверного взаимодействия.
</div>

</div>


## Текущий AI при отладке

Откройте корень `ai-service`, выполните `uv sync --locked` и выберите `.venv` как Python interpreter. Модуль запуска — `orbit_ai`, не Uvicorn/FastAPI. Настройте локальный `.env`, примените миграции и не поднимайте второй процесс на уже занятом `50057`. Для изолированного backend debug достаточно [зависимостей infra](../how-to/ai-local-stack.md). Tasks/Flutter в общей конфигурации выше — учебный/целевой пример, не готовый production-клиент.
