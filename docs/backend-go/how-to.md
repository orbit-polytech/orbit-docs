# Рецепты: «Хочу добавить X, куда писать?»

Когда перед вами стоит задача в Go-сервисе (`auth-service`, `users-service`, `schedule-service`, `gateway-service`), ориентируйтесь на эту матрицу.

---

## 1. Навигационная матрица изменений

| Что я хочу сделать | В какие файлы и папки писать код |
|---|---|
| **Добавить новое поле к сущности** (например, в профиль пользователя) | 1. Поле и валидация в `internal/domain/` и `internal/usecase/`.<br>2. SQL-миграция `migrations/0000N_<name>.sql` (`-- +goose Up` / `-- +goose Down`).<br>3. Маппинг в `internal/infrastructure/postgres/`.<br>4. Поле в `.proto` в репозитории `contracts`.<br>5. Маппинг DTO в `internal/transport/grpc/` (и в `gateway-service` в `internal/transport/http/v1/`). |
| **Добавить новое бизнес-правило** | 1. Проверка/инвариант в методе доменной сущности в `internal/domain/`.<br>2. Новая доменная ошибка в `internal/domain/errors.go`.<br>3. Модульный юнит-тест в `internal/domain/*_test.go`. |
| **Реализовать новый сценарий** | 1. Новый файл в `internal/usecase/`.<br>2. Юнит-тест с моками в `internal/usecase/*_test.go`.<br>3. Вызов сценария из `internal/transport/grpc/` и связывание в `internal/app/`. |
| **Добавить новый RPC-метод** | 1. Объявить метод в `.proto` в репозитории `contracts` и выпустить тег.<br>2. Обновить зависимость `github.com/orbit-polytech/contracts` в `go.mod`.<br>3. Реализовать метод в `internal/transport/grpc/`. |
| **Написать новый SQL-запрос к базе** | 1. Метод в репозитории в `internal/infrastructure/postgres/`.<br>2. Тест репозитория (unit/integration). |
| **Добавить кэширование или rate-limit в Redis** | 1. Интерфейс в `internal/usecase/ports.go` (или инфраструктурный клиент в `gateway-service`).<br>2. Реализация на `go-redis` в `internal/infrastructure/redisstore/` (или `internal/infrastructure/ratelimit/`). |
| **Добавить новую переменную окружения** | 1. Новое поле, чтение и валидация в `internal/config/`.<br>2. Пример значения в `.env.example` сервиса и в `infra/.env.example` / `infra/compose.yaml`. |
| **Добавить HTTP-middleware или gRPC-интерцептор** | `internal/transport/http/middleware/` (в Gateway) или `internal/transport/grpc/` (в доменном сервисе) + подключение в `internal/app/`. |
| **Добавить маппинг ошибки в gRPC-статус / HTTP** | `internal/transport/grpc/` (маппинг доменных ошибок в `codes.*`) или `internal/transport/http/httputil/grpc_error.go` в Gateway. |
| **Запустить сборку, тесты или линтер** | Команды в `Taskfile.yml`: `task build`, `task test`, `task lint`, `task run`. |

---

## 2. Типовые сценарии

### Как добавить SQL-миграцию (Goose)
Во всех Go-сервисах с PostgreSQL используется **`goose`** и единый `.sql`-файл на миграцию:

```bash
# Через Taskfile.yml (например, в auth-service)
task migrate-create NAME=add_display_name

# Либо напрямую через goose:
goose -dir ./migrations -s create add_display_name sql
```

Пример файла `migrations/00002_add_display_name.sql`:
```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE profiles ADD COLUMN display_name VARCHAR(100) NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE profiles DROP COLUMN IF EXISTS display_name;
-- +goose StatementEnd
```

!!! important "Проверка отката"
    Секция `-- +goose Down` обязана полностью и без ошибок откатывать всё, что создала секция `-- +goose Up`. Проверяйте применение и откат локально через `task migrate-up` и `task migrate-down`.

---

### Как протестировать usecase без поднятия базы данных
Благодаря портам в `internal/usecase/ports.go`, в юнит-тестах `internal/usecase/*_test.go` достаточно передать лёгкую in-memory заглушку:

```go
type mockProfileRepo struct {
	saved *domain.Profile
}

func (m *mockProfileRepo) Save(ctx context.Context, p *domain.Profile) error {
	m.saved = p
	return nil
}

func TestUpdateProfile(t *testing.T) {
	repo := &mockProfileRepo{}
	uc := usecase.NewUpdateProfileUseCase(repo)

	err := uc.Execute(context.Background(), "acc-1", "Иван")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if repo.saved == nil {
		t.Fatal("профиль не был передан в репозиторий")
	}
}
```
Тест выполняется за доли миллисекунды, не требует Docker и не зависит от сети.
