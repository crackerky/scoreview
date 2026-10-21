package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDummyHashUsesCurrentParams は、存在しないユーザーで照合するダミーのハッシュが現在の
// パラメータで作られていることを確かめる(違うと処理時間が揃わず、アドレスの有無が漏れる)。
func TestDummyHashUsesCurrentParams(t *testing.T) {
	p, salt, key, err := decodePHC(dummyHash)
	require.NoError(t, err)
	assert.Equal(t, params{memory: argonMemoryKiB, time: argonTime, threads: argonThreads}, p)
	assert.Len(t, salt, argonSaltLen)
	assert.Len(t, key, int(argonKeyLen))
}
