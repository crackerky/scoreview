package talk

import (
	"context"
	"strings"

	"scoreview/api/internal/entry"
)

// Notifier は申込の出来事を、代表に公式アカウントから知らせる(entry.Notifier)。返信のトークンが無いので
// push で送り、公式アカウントの月の配信数に数えられる。
type Notifier struct {
	store     Store
	messenger Messenger
	siteURL   string
}

var _ entry.Notifier = (*Notifier)(nil)

// NewNotifier は Notifier を返す。siteURL は承認の知らせに付けるスコアサイトの URL(末尾の / は無くてよい)。
func NewNotifier(store Store, messenger Messenger, siteURL string) *Notifier {
	return &Notifier{store: store, messenger: messenger, siteURL: strings.TrimRight(siteURL, "/")}
}

// Notify は entry.Notifier.Notify。ブロックしている人には送らない(送っても届かず、配信数だけが減る)。
func (n *Notifier) Notify(ctx context.Context, notice entry.Notice) error {
	recipient, err := n.store.Recipient(ctx, notice.LineUserID)
	if err != nil {
		return err
	}
	if !recipient.Friend {
		return nil
	}
	messages := NoticeMessages(notice, n.siteURL)
	if len(messages) == 0 {
		return nil
	}
	return n.messenger.Push(ctx, recipient.UserID, messages)
}
