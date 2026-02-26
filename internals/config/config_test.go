package config

import (
	"os"
	"testing"

	"catgoose/go-htmx-demo/internals/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {
	ResetForTesting()

	os.Setenv("SERVER_LISTEN_PORT", "9090")
	defer os.Unsetenv("SERVER_LISTEN_PORT")

	config, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, "9090", config.ServerPort)

	// Subsequent calls return the same instance
	config2, err := GetConfig()
	require.NoError(t, err)
	assert.Equal(t, config, config2)
}

func TestMustGetConfig(t *testing.T) {
	ResetForTesting()

	os.Setenv("SERVER_LISTEN_PORT", "7070")
	defer os.Unsetenv("SERVER_LISTEN_PORT")

	config := MustGetConfig()
	assert.Equal(t, "7070", config.ServerPort)
}

func TestMustGetConfigPanic(t *testing.T) {
	ResetForTesting()
	logger.Init()

	os.Unsetenv("SERVER_LISTEN_PORT")

	assert.Panics(t, func() {
		MustGetConfig()
	})
}

func TestConfigSingleton(t *testing.T) {
	ResetForTesting()

	os.Setenv("SERVER_LISTEN_PORT", "1234")
	defer os.Unsetenv("SERVER_LISTEN_PORT")

	config1, err := GetConfig()
	require.NoError(t, err)

	config2, err := GetConfig()
	require.NoError(t, err)

	config3 := MustGetConfig()

	assert.Equal(t, config1, config2)
	assert.Equal(t, config1, config3)
}

func TestAppConfigFields(t *testing.T) {
	config := &AppConfig{
		ServerPort: "5555",
	}

	assert.Equal(t, "5555", config.ServerPort)
}
