package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/domain"
	"scoreview/api/internal/store"
)

// 閲覧用の API は、非公開のグループを ID の直接指定でも出さない
// (docs/requirements.md「権限モデル」。セキュリティ点検 L-6)。
func TestBrowseHidesUnpublished(t *testing.T) {
	db, _, sc := setup(t)
	ctx := context.Background()

	_, err := store.GetGroup(ctx, db, sc.Hidden.ID)
	var notFound *domain.NotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, domain.ResourceGroup, notFound.Resource)

}
