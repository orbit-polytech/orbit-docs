package config_test

import (
	"os"
	"testing"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --8<-- [start:config_tests]
func TestConfig_Load_Defaults(t *testing.T) {
	_ = os.Unsetenv("GRPC_PORT")
	_ = os.Unsetenv("APP_ENV")
	_ = os.Unsetenv("LOG_LEVEL")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 50051, cfg.GRPCPort)
	assert.Equal(t, config.EnvDevelopment, cfg.Env)
	assert.Equal(t, config.LogLevelInfo, cfg.LogLevel)
}

func TestConfig_Load_CustomValid(t *testing.T) {
	t.Setenv("GRPC_PORT", "9090")
	t.Setenv("APP_ENV", "production")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 9090, cfg.GRPCPort)
	assert.Equal(t, config.EnvProduction, cfg.Env)
	assert.Equal(t, config.LogLevelDebug, cfg.LogLevel)
}

func TestConfig_Load_InvalidPort(t *testing.T) {
	t.Setenv("GRPC_PORT", "999999")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "некорректный GRPC_PORT")
}

func TestConfig_Load_InvalidEnv(t *testing.T) {
	t.Setenv("APP_ENV", "unknown-cloud")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "недопустимый APP_ENV")
}

func TestConfig_Load_InvalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "verbose")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "недопустимый LOG_LEVEL")
}

// --8<-- [end:config_tests]
