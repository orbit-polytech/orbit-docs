# Урок 4: OCP в обработке и маппинге ошибок

## 1. Как было сделано изначально (Бесконечный Switch)

В первой версии функции `mapError` маппинг доменных ошибок в gRPC статусы был реализован гигантским блоком `switch-case`, где каждая ошибка перечислялась по отдельности:

```go
// ПЛОХО: Монолитный switch
func mapError(err error) error {
    switch {
    case errors.Is(err, domain.ErrEmptyTaskID):
        return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, domain.ErrEmptyTitle):
        return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, domain.ErrTitleTooShort):
        return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, domain.ErrTitleTooLong):
        return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, domain.ErrDescriptionTooLong):
        return status.Error(codes.InvalidArgument, err.Error())
    // ... еще 20 строк ...
    }
}
```

---

## 2. В чём скрытый дефект

Нарушение **Open-Closed Principle (OCP)**:
> «Программные сущности должны быть открыты для расширения, но закрыты для модификации.»

Каждый раз, когда разработчик добавлял новое доменное правило (например, `ErrInvalidPriority`), он был **обязан лезть в транспортный пакет** и дописывать новую ветку `case`. Если разработчик забывал это сделать, ошибка попадала в секцию `default` и клиент получал `500 Internal Server Error` вместо понятной `400 Bad Request`.

---

## 3. Как правильно делать в реальном сервисе

Есть два зрелых архитектурных подхода:

### Подход А: Классификация ошибок по категориям (Domain Error Kind)
Доменные ошибки оборачиваются или реализуют интерфейс с типом ошибки:

```go
type ErrorKind int

const (
    KindValidation ErrorKind = iota
    KindNotFound
    KindConflict
    KindPermission
)

type DomainError struct {
    Kind ErrorKind
    Msg  string
}
```

Транспортный маппер проверяет категорию `Kind`:
```go
func mapError(err error) error {
    var de DomainError
    if errors.As(err, &de) {
        switch de.Kind {
        case KindValidation:
            return status.Error(codes.InvalidArgument, de.Msg)
        case KindNotFound:
            return status.Error(codes.NotFound, de.Msg)
        case KindConflict:
            return status.Error(codes.Aborted, de.Msg)
        }
    }
    return status.Error(codes.Internal, "внутренняя ошибка")
}
```
Теперь добавление 10 новых ошибок валидации не требует изменения ни строчки кода в транспорте!

### Подход Б: Группировка через множественный `errors.Is` в блоке `case`
Если сохраняется плоский список ошибок, они группируются по возвращаемому gRPC-коду:

```go
case errors.Is(err, domain.ErrEmptyTaskID),
     errors.Is(err, domain.ErrEmptyTitle),
     errors.Is(err, domain.ErrTitleTooShort),
     errors.Is(err, domain.ErrInvalidDeadline):
    return status.Error(codes.InvalidArgument, err.Error())
```
Код становится в 5 раз короче, читаемее и проще в аудите.
