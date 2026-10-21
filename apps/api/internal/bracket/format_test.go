package bracket_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scoreview/api/internal/bracket"
	"scoreview/api/internal/domain"
	"scoreview/api/internal/format"
)

func TestWithFormatTypes(t *testing.T) {
	registry := bracket.NewRegistry(bracket.WithFormatTypes())
	var ids []string
	for _, tmpl := range registry.List() {
		ids = append(ids, tmpl.ID)
		plan := tmpl.Plan()
		require.NoError(t, plan.Layout.Validate())
		assert.Equal(t, tmpl.Teams, plan.Teams)
	}
	assert.Equal(t, []string{"t7", "t8_all_places", "t16"}, ids)
	t8, ok := registry.Get("t8_all_places")
	require.True(t, ok)
	assert.Len(t, t8.Plan().Matches, 12, "参照サイトの T8 全順位型(R3 4 章)")
}

func TestFormatTemplateUnknownType(t *testing.T) {
	_, err := bracket.FormatTemplate("x", format.BracketType("T5"), domain.I18nText{})
	assert.ErrorIs(t, err, domain.ErrBadRequest)
}
