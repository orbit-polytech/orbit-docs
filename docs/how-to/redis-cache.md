# Как работать с Redis кэшем

В микросервисах Orbit AI оперативное in-memory хранилище **Redis 7** используется для кэширования часто запрашиваемых и медленно меняющихся данных (например, расписания групп в `schedule-service`, справочников аудиторий в `navigation-service`, а также для блэклиста отозванных JWT-токенов).

Для надежной работы кэша под нагрузкой применяется классический паттерн **Cache-Aside (Lazy Loading)** в сочетании с библиотекой **`singleflight`** (защита от эффекта толпы / Cache Stampede) и **TTL Jitter** (защита от лавины инвалидации).

---

## 1. Конвенция именования ключей в Redis

Все ключи в Redis должны следовать строгому шаблону с двоеточиями в качестве разделителей пространств имён:

$$\text{service\_name} : \text{entity\_type} : \text{identifier}$$

| Назначение | Пример ключа | TTL |
| :--- | :--- | :--- |
| Расписание учебной группы | `schedule:group:261-334` | 30 минут + Jitter |
| Справочник аудитории корпуса | `navigation:room:БС-А401` | 24 часа |
| Отозванный токен (JTI) | `auth:blacklist:018f2345-6789` | До истечения JWT `exp` |
| Счётчик попыток входа (Rate limit) | `gateway:ratelimit:ip:192.168.1.5` | 1 минута |

---

## 2. Реализация паттерна Cache-Aside с `singleflight`

Когда сотни студентов одновременно открывают мобильное приложение в 8:59 утра, истечение срока жизни кэша для группы `261-334` может привести к **Cache Stampede (эффекту собачьей своры / Dogpile)**: все 100 запросов одновременно пойдут в базу данных, перегружая пул соединений PostgreSQL.

Инструмент `singleflight.Group` из стандартного расширения `golang.org/x/sync/singleflight` гарантирует, что для одного и того же ключа в БД пойдет **ровно один запрос**, а все остальные горутины дождутся его ответа.

### Пример безопасного кэш-репозитория

```go
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"github.com/orbit-polytech/schedule-service/internal/domain"
)

type CachedScheduleRepository struct {
	rdb        *redis.Client
	dbRepo     domain.ScheduleRepository
	sfGroup    singleflight.Group
	baseTTL    time.Duration
}

func NewCachedScheduleRepository(
	rdb *redis.Client, 
	dbRepo domain.ScheduleRepository, 
	baseTTL time.Duration,
) *CachedScheduleRepository {
	return &CachedScheduleRepository{
		rdb:     rdb,
		dbRepo:  dbRepo,
		baseTTL: baseTTL,
	}
}

func (r *CachedScheduleRepository) GetByGroup(ctx context.Context, groupID string) (*domain.Schedule, error) {
	cacheKey := fmt.Sprintf("schedule:group:%s", groupID)

	// 1. Попытка чтения из Redis кэша
	val, err := r.rdb.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var schedule domain.Schedule
		if err := json.Unmarshal(val, &schedule); err == nil {
			return &schedule, nil // Cache Hit (попадание в кэш, ~1-2 мс)
		}
	} else if !errors.Is(err, redis.Nil) {
		// Ошибка Redis не должна ломать работу сервиса: логируем и продолжаем чтение из БД
		// Fallback to Database
	}

	// 2. Cache Miss: Используем singleflight для исключения параллельных дублирующих запросов в БД
	result, err, _ := r.sfGroup.Do(cacheKey, func() (any, error) {
		// Чтение из медленного источника (PostgreSQL)
		schedule, err := r.dbRepo.GetByGroup(ctx, groupID)
		if err != nil {
			return nil, err
		}

		// Сериализация и сохранение в Redis
		payload, err := json.Marshal(schedule)
		if err == nil {
			// Добавляем TTL Jitter (случайный разброс от 0 до 60 секунд),
			// чтобы ключи не инвалидировались одновременно (Cache Avalanche)
			jitter := time.Duration(rand.IntN(60)) * time.Second
			totalTTL := r.baseTTL + jitter

			_ = r.rdb.Set(ctx, cacheKey, payload, totalTTL).Err()
		}

		return schedule, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*domain.Schedule), nil
}

// 3. Инвалидация кэша при обновлении расписания диспетчером
func (r *CachedScheduleRepository) Invalidate(ctx context.Context, groupID string) error {
	cacheKey := fmt.Sprintf("schedule:group:%s", groupID)
	return r.rdb.Del(ctx, cacheKey).Err()
}
```

---

## 3. Сравнение: Как нельзя и Как правильно

<div class="grid" markdown>

<div class="card" markdown>
### :x: Как нельзя

- **Падать с ошибкой, если Redis временно недоступен.** Кэш — это уровень оптимизации, а не источник истины (Source of Truth). При сбое Redis сервис должен продолжать отдавать данные напрямую из PostgreSQL.
- **Ставить всем ключам жесткий одинаковый TTL (например, ровно 1 час).** Если в 12:00 было загружено 500 расписаний, в 13:00 они все синхронно исчезнут, создав скачкообразный удар по базе данных (Cache Avalanche).
- **Использовать бесконечный TTL (`0`) без механизма явной инвалидации.** Студенты будут видеть устаревшие аудитории и старое расписание даже после правок в деканате.
- **Хранить в Redis ключи без префикса (например, просто ID `261-334`).** Произойдет пересечение ключей между разными модулями (коллизия имён).
</div>

<div class="card" markdown>
### :white_check_mark: Как правильно

- **Использовать паттерн Cache-Aside с fallback на БД** при любой ошибке сетевого клиента Redis.
- **Защищаться от Cache Stampede через `singleflight.Group`** при частых параллельных запросах одного и того же ключа.
- **Добавлять случайный разброс времени жизни ключа (TTL Jitter):** `TTL = baseTTL + rand(0..Jitter)`.
- **Строго следовать конвенции ключей `service:entity:id`.**
- **Выполнять инвалидацию (`Del`) сразу при коммите изменений в базе данных.**
</div>

</div>
