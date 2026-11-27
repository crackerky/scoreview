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

// TextJoinHowTo は「参加」と送ったが招待コードが読めないときの返信(招待リンクを開かずに打った、など)。
const TextJoinHowTo = "メンバー登録は、チームの代表から届いた招待リンクを開いて、そのまま参加のメッセージを送ってください。\n" +
	"招待リンクは、チームの代表がチームの LINE グループなどに送っています。"

// TextHelp は知らない言葉への使い方の案内(同じ人に 1 日 1 回だけ返す)。
const TextHelp = "このトークでできること\n\n" +
	"大会への申込とメンバー登録\n「エントリー」と送るか、メニューの「エントリー」を押して、代表かメンバーかを選んでください。\n\n" +
	"記録係の記録\n「記録」と送ってください。\n\n" +
	"ほかのご用件は、このままメッセージを送ってください。運営が確かめます。"

// TextNoOpenTournament は申込を受け付けている大会も、出した申込も無いときの、代表の方への返信。
const TextNoOpenTournament = "いま申込を受け付けている大会はありません。"

// TeamLink は返信に並べる、大会ごとの申込のリンク。
type TeamLink struct {
	Tournament string
	URL        string
}

// JoinedTeamLink は返信に並べる、メンバーとして登録したチームの登録のリンク。
type JoinedTeamLink struct {
	Team       string
	Tournament string
	URL        string
}

// TextMemberHowTo は、まだどのチームにもメンバーとして登録していない人への、登録のしかた。
const TextMemberHowTo = "メンバー登録は、チームの代表が申し込んだあとにできます。\n" +
	"代表から届く招待リンクを開いて、そのまま「参加」のメッセージを送ってください。名前は自分で入れます。"

// TextEntryChoice は「エントリー」への返信。代表とメンバーで入口が違うので、本人に選んでもらう。
const TextEntryChoice = "チームの代表の方は、大会への申込から始めてください。\n" +
	"メンバーの方は、代表が申し込んだあとに、代表から届く招待リンクで登録します。\n\n" +
	"どちらか選んでください。"

// EntryChoiceMessage は「エントリー」への返信の、代表かメンバーかを選ぶボタンつきの文。
func EntryChoiceMessage() Message {
	return Message{Text: TextEntryChoice, Choices: []Choice{
		{Label: "代表として申し込む", Data: PostbackEntryRep},
		{Label: "メンバーとして登録する", Data: PostbackEntryMember},
	}}
}

// linkNote はリンクの扱いの一文。
func linkNote() string {
	return fmt.Sprintf("リンクはあなた専用で、%d 日間使えます。ほかの人に送らないでください。", entry.LinkLifetimeDays)
}

// RepReplyText は代表の入口への返信。大会ごとの申込のリンクを並べる。メンバーの名前は分かる人だけ書けば
// よく、申し込むとメンバーに送る招待リンクが届くことを先に書く(メンバー表を埋めないと申し込めないと
// 思わせないため)。受付を締めた大会でも、申込を出した人には、出した申込を確かめるリンクが並ぶ。
func RepReplyText(teams []TeamLink) string {
	if len(teams) == 0 {
		return TextNoOpenTournament
	}
	var text strings.Builder
	text.WriteString("代表の方の申込は、大会のリンクから入れます。出した申込の確かめと直しも、同じリンクからできます。\n" +
		"メンバーの名前は、分かれば書いてください。書かなくても申し込めます。\n" +
		"申し込むと、メンバーに送る招待リンクが届きます。")
	for _, link := range teams {
		fmt.Fprintf(&text, "\n\n%s\n%s", link.Tournament, link.URL)
	}
	text.WriteString("\n\n" + linkNote())
	return text.String()
}

// MemberReplyText はメンバーの入口への返信。登録したチームがあればその確かめと直しのリンクを、無ければ
// 登録のしかたを返す。
func MemberReplyText(joined []JoinedTeamLink) string {
	if len(joined) == 0 {
		return TextMemberHowTo
	}
	var text strings.Builder
	text.WriteString("登録したチームの確かめと直しは、チームのリンクからできます。")
	for _, link := range joined {
		// チーム名は代表が書いたものなので「」で囲む(MemberLinkText と同じ)
		fmt.Fprintf(&text, "\n\n「%s」%s\n%s", link.Team, link.Tournament, link.URL)
	}
	text.WriteString("\n\nほかのチームに登録するときは、そのチームの代表から届く招待リンクを開いてください。\n" + linkNote())
	return text.String()
}

// TextNotScorer は記録係に選ばれていない人が「記録」と送ったときの返信。
const TextNotScorer = "記録係に選ばれていません。\nチームの代表の方に、記録係に選んでもらってください。"

// ScorerLinksText は「記録」への返信。記録係に選ばれたチームごとに、記録の画面に入るリンクを並べる。
func ScorerLinksText(links []JoinedTeamLink) string {
	if len(links) == 0 {
		return TextNotScorer
	}
	var text strings.Builder
	text.WriteString("記録の画面は、チームのリンクから開きます。自分たちの試合のスコアを入れてください。")
	for _, link := range links {
		// チーム名は代表が書いたものなので「」で囲む(MemberLinkText と同じ)
		fmt.Fprintf(&text, "\n\n「%s」%s\n%s", link.Team, link.Tournament, link.URL)
	}
	fmt.Fprintf(&text, "\n\nリンクはあなた専用で、%d 日間使えます。ほかの人に送らないでください。",
		entry.LinkLifetimeDays)
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
			"下のボタンを押すと、直す画面のリンクが届きます。", team, detail.ReturnNote),
			Choices: []Choice{{Label: "直す画面を開く", Data: PostbackEntryRep}}}}
	case entry.NoticeScorerChosen:
		return []Message{{Text: fmt.Sprintf("%sの%sの記録係になりました。\n"+
			"当日は、このトークで「記録」と送ると、記録の画面が開きます。自分たちの試合のスコアを入れてください。",
			team, tournament)}}
	case entry.NoticeReopened:
		return []Message{{Text: fmt.Sprintf("%sの%sへの申込は、運営の確認待ちに戻りました。\n"+
			"運営が確かめたら、このトークでお知らせします。", team, tournament)}}
	case entry.NoticeCancelled:
		return []Message{{Text: fmt.Sprintf("%sの%sへの申込は取り消されました。", team, tournament)}}
	}
	return nil
}
