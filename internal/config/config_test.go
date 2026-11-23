package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var keys = []string{
	"APP_ENV", "APP_PORT", "APP_KEY", "ADMIN_USERNAME", "ADMIN_PASSWORD",
	"DB_CONNECTION", "DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD", "DB_SSLMODE",
	"SEARCH_ENGINE_ENDPOINT", "OPENAI_ENDPOINT", "OPENAI_KEY", "OPENAI_MODEL",
}

func TestLoadDefaults(t *testing.T) {
	for _, k := range keys {
		t.Setenv(k, "")
	}
	c := Load()
	assert.Equal(t, "development", c.AppEnv)
	assert.Equal(t, "8081", c.AppPort)
	assert.Equal(t, "admin", c.AdminUsername)
	assert.Equal(t, "pabwe_p9_blog", c.DBDatabase)
	assert.Equal(t, "https://delcom.org/api/v1/search", c.SearchEngineEndpoint)
	assert.Equal(t, "", c.OpenAIKey)
	assert.Equal(t,
		"host=127.0.0.1 port=5432 user=postgres password=postgres dbname=pabwe_p9_blog sslmode=disable TimeZone=Asia/Jakarta",
		c.DSN())
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("APP_PORT", "9000")
	t.Setenv("OPENAI_MODEL", "gpt-x")
	t.Setenv("DB_HOST", "db.local")
	c := Load()
	assert.Equal(t, "9000", c.AppPort)
	assert.Equal(t, "gpt-x", c.OpenAIModel)
	assert.Contains(t, c.DSN(), "host=db.local")
}
