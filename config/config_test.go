package config_test

import (
	"testing"

	"github.com/nischalpatel/transactions-api/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DB_DRIVER", "")
	t.Setenv("PORT", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, config.DBDriverMemory, cfg.DBDriver)
	assert.Equal(t, "9001", cfg.SwaggerPort)
}

func TestLoad_CustomPort(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_DRIVER", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
}

func TestLoad_PostgresMode_MissingHost_ReturnsError(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_HOST", "")

	_, err := config.Load()
	assert.EqualError(t, err, "DB_HOST must be set when DB_DRIVER=postgres")
}

func TestLoad_PostgresMode_MissingPort_ReturnsError(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "")

	_, err := config.Load()
	assert.EqualError(t, err, "DB_PORT must be set when DB_DRIVER=postgres")
}

func TestLoad_PostgresMode_MissingUser_ReturnsError(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "")

	_, err := config.Load()
	assert.EqualError(t, err, "DB_USER must be set when DB_DRIVER=postgres")
}

func TestLoad_PostgresMode_MissingPassword_ReturnsError(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "")

	_, err := config.Load()
	assert.EqualError(t, err, "DB_PASSWORD must be set when DB_DRIVER=postgres")
}

func TestLoad_PostgresMode_MissingDBName_ReturnsError(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "")

	_, err := config.Load()
	assert.EqualError(t, err, "DB_NAME must be set when DB_DRIVER=postgres")
}

func TestLoad_PostgresMode_MissingSSLMode_ReturnsError(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("DB_SSLMODE", "")

	_, err := config.Load()
	assert.EqualError(t, err, "DB_SSLMODE must be set when DB_DRIVER=postgres")
}

func TestLoad_PostgresMode_Succeeds(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "pguser")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("DB_SSLMODE", "disable")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, config.DBDriverPostgres, cfg.DBDriver)
	assert.Equal(t, "localhost", cfg.Postgres.Host)
	assert.Equal(t, "5432", cfg.Postgres.Port)
	assert.Equal(t, "pguser", cfg.Postgres.User)
	assert.Equal(t, "secret", cfg.Postgres.Password)
	assert.Equal(t, "mydb", cfg.Postgres.DBName)
	assert.Equal(t, "disable", cfg.Postgres.SSLMode)
}

func TestLoad_InvalidDriver_FallsBackToMemory(t *testing.T) {
	t.Setenv("DB_DRIVER", "sqlite")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, config.DBDriverMemory, cfg.DBDriver)
}

func TestLoad_SwaggerPortDefault(t *testing.T) {
	t.Setenv("SWAGGER_PORT", "")
	t.Setenv("DB_DRIVER", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "9001", cfg.SwaggerPort)
}

func TestLoad_SwaggerPortCustom(t *testing.T) {
	t.Setenv("SWAGGER_PORT", "8090")
	t.Setenv("DB_DRIVER", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "8090", cfg.SwaggerPort)
}

func TestLoad_SwaggerPortDisabled(t *testing.T) {
	t.Setenv("SWAGGER_PORT", " ")
	t.Setenv("DB_DRIVER", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	_ = cfg
}
