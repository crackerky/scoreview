package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/config"
)

const dsn = "postgres://localhost/scoreview"

func TestLoadAPIRejectsPlaceholderToken(t *testing.T) {
	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("PORT", "")
	// 以前の .env.example の値。長さは足りているが、公開されている値なので起動しない
	t.Setenv("INTERNAL_API_TOKEN", "dev-internal-token-change-me-0123456789")
	_, err := config.LoadAPI()
	require.ErrorIs(t, err, config.ErrMissingEnv)
	t.Setenv("INTERNAL_API_TOKEN", "DEV-INTERNAL-TOKEN-CHANGE-ME-0123456789")
	_, err = config.LoadAPI()
	require.ErrorIs(t, err, config.ErrMissingEnv)
}

func TestLoadAPI(t *testing.T) {
	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("PORT", "")
	t.Setenv("INTERNAL_API_TOKEN", "short")
	_, err := config.LoadAPI()
	require.ErrorIs(t, err, config.ErrMissingEnv)

	t.Setenv("INTERNAL_API_TOKEN", "0123456789abcdef0123456789abcdef")
	cfg, err := config.LoadAPI()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
}
