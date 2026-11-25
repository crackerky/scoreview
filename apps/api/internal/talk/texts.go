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

// InviteText はメンバーへの登録のお願い。代表がそのままチームの LINE グループに転送できる形にする(web の
// 「LINE でメンバーに登録のお願いを送る」と同じ文)。
func InviteText(team, inviteURL string) string {
	return fmt.Sprintf("%s のメンバー登録のお願いです。\n下のリンクを開いて、そのまま「参加」のメッセージを送ってください。\n%s",
		team, inviteURL)
}

// NoticeMessages は代表への知らせの文(entry.Notice)。受付には、メンバーへの登録のお願いを別の吹き出しで
// 付ける(その吹き出しだけを転送できるように。1 回の送信は吹き出しが 5 つまで、配信数は 1 通)。知らせる
// 出来事でなければ空。
func NoticeMessages(notice entry.Notice, siteURL string) []Message {
	detail := notice.Entry
	team := "「" + detail.FullName + "」"
	tournament := "「" + notice.Tournament.Name + "」"
	switch notice.Kind {
	case entry.NoticeSubmitted:
		text := fmt.Sprintf("%sの申込を受け付けました。\n大会: %s\n部門: %s\n\n運営が確かめたら、このトークでお知らせします。",
			team, notice.Tournament.Name, detail.Division.Name.Ja)
		if detail.InviteURL == "" {
			return []Message{{Text: text}}
		}
		text += "\n\nメンバーには、次のメッセージをチームの LINE グループに転送して、登録をお願いしてください。"
		return []Message{{Text: text}, {Text: InviteText(detail.FullName, detail.InviteURL)}}
	case entry.NoticeResubmitted:
		return []Message{{Text: fmt.Sprintf("%sの直した申込を受け付けました。\n運営が確かめたら、このトークでお知らせします。",
			team)}}
	case entry.NoticeApproved:
		return []Message{{Text: fmt.Sprintf("%sの%sへの申込を受け付け、参加が決まりました。\n日程と結果は、スコアサイトで見られます。\n%s/tournaments/%d",
			team, tournament, siteURL, notice.Tournament.ID)}}
	case entry.NoticeReturned:
		return []Message{{Text: fmt.Sprintf("%sの申込について、運営から直してほしいことがあります。\n\n%s\n\n"+
			"メニューの「エントリー」を押すか「エントリー」と送ると、直す画面が開きます。", team, detail.ReturnNote)}}
	case entry.NoticeCancelled:
		return []Message{{Text: fmt.Sprintf("%sの%sへの申込は取り消されました。", team, tournament)}}
	}
	return nil
}
