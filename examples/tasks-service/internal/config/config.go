package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// --8<-- [start:config_struct]
// Config описывает параметры запуска микросервиса.
type Config struct {
	GRPCPort int
	Env      string
	LogLevel string
}

const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"

	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// Load загружает конфигурацию из переменных окружения с валидацией допустимых значений.
func Load() (*Config, error) {
	portStr := getEnv("GRPC_PORT", "50051")
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return nil, fmt.Errorf("некорректный GRPC_PORT %q: ожидается порт от 1 до 65535", portStr)
	}

	env := strings.ToLower(strings.TrimSpace(getEnv("APP_ENV", EnvDevelopment)))
	switch env {
	case EnvDevelopment, EnvStaging, EnvProduction:
		// валидное окружение
	default:
		return nil, fmt.Errorf("недопустимый APP_ENV %q: допустимы development, staging, production", env)
	}

	logLevel := strings.ToLower(strings.TrimSpace(getEnv("LOG_LEVEL", LogLevelInfo)))
	switch logLevel {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		// валидный уровень логирования
	default:
		return nil, fmt.Errorf("недопустимый LOG_LEVEL %q: допустимы debug, info, warn, error", logLevel)
	}

	return &Config{
		GRPCPort: port,
		Env:      env,
		LogLevel: logLevel,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// --8<-- [end:config_struct]
