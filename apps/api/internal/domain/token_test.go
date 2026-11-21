package domain_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
)

func TestNewToken(t *testing.T) {
	token, err := domain.NewToken(bytes.NewReader(make([]byte, domain.TokenBytes)))
	require.NoError(t, err)
	assert.Equal(t, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", token)

	_, err = domain.NewToken(bytes.NewReader(make([]byte, domain.TokenBytes-1)))
	require.Error(t, err, "a short random source")
}

func TestHashToken(t *testing.T) {
	assert.Len(t, domain.HashToken("a"), 32)
	assert.Equal(t, domain.HashToken("a"), domain.HashToken("a"))
	assert.NotEqual(t, domain.HashToken("a"), domain.HashToken("b"))
}
