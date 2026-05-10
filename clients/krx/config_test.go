package krx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequiresAuthKey(t *testing.T) {
	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth key is required")
}

func TestNewRejectsNilOption(t *testing.T) {
	_, err := New(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "option is required")
}

func TestWithSampleBaseURLSelectsSampleEndpoint(t *testing.T) {
	client, err := New(
		WithAuthKey("test-auth-key"),
		WithSampleBaseURL("http://127.0.0.1/sample/"),
	)
	require.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1/sample", client.http.BaseURL)
}
