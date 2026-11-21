package entry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scoreview/api/internal/entry"
)

func TestProfileUpdates(t *testing.T) {
	rep := entry.RepProfile(entry.Draft{RepName: "山田 太郎", RepPhone: "090", RepEmail: "a@example.com"})
	assert.Equal(t, entry.ProfileUpdate{Name: "山田 太郎",
		Contact: &entry.Contact{Phone: "090", Email: "a@example.com"}}, rep)
	member := entry.MemberProfile(entry.JoinDraft{Name: "佐藤 花子", Number: "12"})
	assert.Equal(t, entry.ProfileUpdate{Name: "佐藤 花子"}, member, "a member registration keeps the contact")
}
