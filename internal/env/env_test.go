package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withEnvFlag temporarily sets the parsed -env flag value for the duration of
// the test, restoring it on cleanup.
func withEnvFlag(t *testing.T, v string) {
	t.Helper()
	prev := *envFlag
	*envFlag = v
	t.Cleanup(func() { *envFlag = prev })
}

func TestResolveMode_EnvFlagWinsOverENV(t *testing.T) {
	withEnvFlag(t, "production")
	t.Setenv("ENV", "staging")

	got := resolveMode()
	assert.Equal(t, "production", got)
}

func TestResolveMode_ENVWhenFlagUnset(t *testing.T) {
	withEnvFlag(t, "")
	t.Setenv("ENV", "production")

	got := resolveMode()
	assert.Equal(t, "production", got)
}

func TestResolveMode_DefaultsToDevelopment(t *testing.T) {
	withEnvFlag(t, "")
	t.Setenv("ENV", "")

	got := resolveMode()
	assert.Equal(t, "development", got)
}

func TestResolveMode_NormalizesShortNames(t *testing.T) {
	withEnvFlag(t, "")
	t.Setenv("ENV", "prod")
	assert.Equal(t, "production", resolveMode())

	t.Setenv("ENV", "dev")
	assert.Equal(t, "development", resolveMode())
}

func TestLookup_DistinguishesEmptyFromMissing(t *testing.T) {
	const key = "DOTHOG_TEST_EMPTY_VS_MISSING"

	// Unset → not present.
	_, ok := Lookup(key)
	assert.False(t, ok, "Lookup of unset key must return ok=false")

	// Explicitly empty → present with empty value.
	t.Setenv(key, "")
	got, ok := Lookup(key)
	assert.True(t, ok, "Lookup of explicitly-empty key must return ok=true")
	assert.Equal(t, "", got)
}

func TestGet_ReturnsValueIncludingEmpty(t *testing.T) {
	const key = "DOTHOG_TEST_GET_EMPTY"

	// Explicitly empty must NOT be treated as missing — Get returns ("", nil).
	t.Setenv(key, "")
	v, err := Get(key)
	require.NoError(t, err, "Get must not error on an explicitly-empty value")
	assert.Equal(t, "", v)
}

func TestGet_ErrorsOnUnsetKey(t *testing.T) {
	const key = "DOTHOG_TEST_GET_MISSING"

	_, err := Get(key)
	require.Error(t, err, "Get must error when the key is unset")
}

func TestGet_ReturnsNonEmptyValue(t *testing.T) {
	const key = "DOTHOG_TEST_GET_VALUE"
	t.Setenv(key, "abc")

	v, err := Get(key)
	require.NoError(t, err)
	assert.Equal(t, "abc", v)
}

func TestGetDefault_ReturnsEmptyWhenSetEmpty(t *testing.T) {
	const key = "DOTHOG_TEST_DEFAULT_EMPTY"
	t.Setenv(key, "")

	got := GetDefault(key, "fallback")
	assert.Equal(t, "", got, "GetDefault must preserve an explicitly-empty value rather than substituting the fallback")
}

func TestGetDefault_ReturnsFallbackWhenUnset(t *testing.T) {
	const key = "DOTHOG_TEST_DEFAULT_UNSET"

	got := GetDefault(key, "fallback")
	assert.Equal(t, "fallback", got)
}

func TestGetDefault_ReturnsValueWhenSet(t *testing.T) {
	const key = "DOTHOG_TEST_DEFAULT_VALUE"
	t.Setenv(key, "abc")

	got := GetDefault(key, "fallback")
	assert.Equal(t, "abc", got)
}
