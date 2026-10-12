# Урок 6: Безопасность рантайма и исключение panic

## 1. Как было сделано изначально (Иллюзия безопасности)

В пакете генератора идентификаторов `idgen` красовался многообещающий комментарий:
*«UUIDGenerator генерирует уникальные идентификаторы без паники.»*

Однако при этом метод `NewEventID()` был написан так:

```go
// ПЛОХО: Вызов паникующего метода
func (g *UUIDGenerator) NewEventID() string {
    return uuid.New().String()
}
```

---

## 2. В чём скрытый баг

Библиотека `github.com/google/uuid` устроена так:
* Метод `uuid.NewRandom()` возвращает `(UUID, error)`.
* Метод `uuid.New()` вызывает `NewRandom()` и **вызывает `panic(err)`**, если криптографический генератор случайных чисел операционной системы (`crypto/rand`) вернул ошибку (например, при исчерпании файловых дескрипторов под высокой нагрузкой или сбое ядра).

Вызов `panic` в Go роняет весь процесс сервиса целиком, если он не перехвачен через `recover()`. Один сбойный запрос валил сервер для всех пользователей.

---

## 3. Как правильно делать в реальном сервисе

1. **Никаких скрытых `panic` в бизнес-логике и утилитах**:
   Все методы генераторов обязаны возвращать явную ошибку:

   ```go
   // ХОРОШО: Возврат явной ошибки
   func (g *UUIDGenerator) NewEventID() (string, error) {
       newUUID, err := uuid.NewRandom()
       if err != nil {
           return "", fmt.Errorf("ошибка криптогенератора при создании UUID: %w", err)
       }
       return newUUID.String(), nil
   }
   ```

2. **Обязательный внешний `RecoveryInterceptor`**:
   Даже при самом аккуратном коде никто не застрахован от разыменования nil-указателя в сторонней библиотеке. Поэтому в gRPC-сервере самым **внешним (первым)** интерцептором ставится перехватчик паник:

   ```go
   func RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
       return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
           defer func() {
               if r := recover(); r != nil {
                   stack := debug.Stack()
                   logger.Error("перехвачена паника в gRPC-хэндлере",
                       slog.Any("recover", r),
                       slog.String("stack", string(stack)),
                   )
                   // Возвращаем клиенту чистую ошибку 500, сервис продолжает жить!
                   err = status.Error(codes.Internal, "внутренняя ошибка сервера")
               }
           }()
           return handler(ctx, req)
       }
   }
   ```
