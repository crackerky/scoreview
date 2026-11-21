package talk

import (
	"fmt"
	"strings"

	"scoreview/api/internal/entry"
)

// 公式アカウントが返す文。運営(ユミさん)が Lイベントで使っていた言い方に近づけ、敬体で短く書く。

// TextInviteNotFound は招待コードのチームが見つからないときの返信。
const TextInviteNotFound = "招待コードが見つかりませんでした。\nチームの代表の方から届いたリンクを、もう一度開いてください。"

// TextTeamCancelled は取り消した申込の招待コードのときの返信。
const TextTeamCancelled = "このチームの申込は取り消されています。\nチームの代表の方に確かめてください。"

// TextNoOpenTournament は申込を受け付けている大会も、出した申込も無いときの返信。
const TextNoOpenTournament = "いま申込を受け付けている大会はありません。"

// TeamLink は返信に並べる、大会ごとの申込のリンク。
type TeamLink struct {
	Tournament string
	URL        string
}

// TeamLinksText は代表の申込のリンクの返信。リンクは本人用で、期限があることを書く。受付を締めた大会
// でも、申込を出した人には、出した申込を確かめるリンクが並ぶ。
func TeamLinksText(links []TeamLink) string {
	if len(links) == 0 {
		return TextNoOpenTournament
	}
	var text strings.Builder
	text.WriteString("大会への申込と、出した申込の確かめは、大会のリンクからできます。\n")
	fmt.Fprintf(&text, "リンクはあなた専用で、%d 日間使えます。ほかの人に送らないでください。",
		entry.LinkLifetimeDays)
	for _, link := range links {
		fmt.Fprintf(&text, "\n\n%s\n%s", link.Tournament, link.URL)
	}
	return text.String()
}

// MemberLinkText はメンバー登録のリンクの返信。チーム名は代表が書いたものなので「」で囲み、公式
// アカウントの文と見分けられるようにする(改行は entry が拒む)。
func MemberLinkText(team, url string) string {
	return fmt.Sprintf("「%s」のメンバー登録はこちらです。\nリンクはあなた専用で、%d 日間使えます。\n\n%s",
		team, entry.LinkLifetimeDays, url)
}
