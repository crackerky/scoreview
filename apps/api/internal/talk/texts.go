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
	"記録係のスコアの入力\n「スコアを入れる」と送るか、メニューの「スコアを入れる」を押してください。\n\n" +
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

// clip は文を n 字までに切る(切ったら最後を「…」にする)。ボタンの文字を LINE の上限に収める。
func clip(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n-1]) + "…"
}

// RepReplyMessage は代表の入口への返信。申し込む大会をボタンで選んでもらう(ボタンは本人用の申込のリンク
// を開く)。メンバーの名前は分かる人だけ書けばよく、申し込むとメンバーに送る招待リンクが届くことを先に
// 書く(メンバー表を埋めないと申し込めないと思わせないため)。受付を締めた大会でも、申込を出した人には、
// 出した申込を確かめるボタンが並ぶ。
func RepReplyMessage(teams []TeamLink) Message {
	if len(teams) == 0 {
		return Message{Text: TextNoOpenTournament}
	}
	choices := make([]Choice, len(teams))
	for i, link := range teams {
		choices[i] = Choice{Label: clip(link.Tournament, MaxChoiceLabel), URL: link.URL}
	}
	return Message{Text: "代表の方の申込です。申し込む大会を選んでください。出した申込の確かめと直しも、同じボタンからできます。\n\n" +
		"メンバーの名前は、分かれば書いてください。書かなくても申し込めます。申し込むと、メンバーに送る招待リンクが届きます。\n\n" +
		linkNote(), Choices: choices}
}

// MemberReplyMessage はメンバーの入口への返信。登録したチームがあればチームのボタン(確かめと直しの
// リンク)を、無ければ登録のしかたを返す。
func MemberReplyMessage(joined []JoinedTeamLink) Message {
	if len(joined) == 0 {
		return Message{Text: TextMemberHowTo}
	}
	choices := make([]Choice, len(joined))
	for i, link := range joined {
		choices[i] = Choice{Label: clip(link.Team+" ・ "+link.Tournament, MaxChoiceLabel), URL: link.URL}
	}
	return Message{Text: "登録したチームの確かめと直しは、チームのボタンからできます。\n" +
		"ほかのチームに登録するときは、そのチームの代表から届く招待リンクを開いてください。\n\n" + linkNote(),
		Choices: choices}
}

// TextNotScorer は記録係に選ばれていない人が「スコアを入れる」を押したときの返信。
const TextNotScorer = "記録係に選ばれていません。\nチームの記録係になるときは、チームの代表の方に選んでもらってください。" +
	"\n\n試合の結果を見るときは、下のボタンからスコアサイトを開けます。"

// ScorerTeamLink は記録係のチームの記録の画面のリンクと、これから記録する試合。
type ScorerTeamLink struct {
	Team       string
	Tournament string
	URL        string
	Matches    []entry.ScorerMatch
}

// MaxScorerTeams は「スコアを入れる」への返信に並べるチームの数(LINE の 1 回の返信は吹き出しが 5 つまで)。
const MaxScorerTeams = 5

// ScorerMessages は「スコアを入れる」への返信。記録係のチームごとに、これから記録する試合をボタンで選んで
// もらう(押すと、その試合のスコアの画面が開く)。試合の一覧のボタンも付ける(時間がずれたときや、終わった
// 試合を直すとき)。記録係でなければ、そのことと、スコアサイトのボタンを返す。
func ScorerMessages(links []ScorerTeamLink, siteURL string) []Message {
	if len(links) == 0 {
		return []Message{{Text: TextNotScorer, Choices: []Choice{{Label: "スコアサイトを見る", URL: siteURL + "/"}}}}
	}
	links = links[:min(len(links), MaxScorerTeams)]
	out := make([]Message, len(links))
	for i, link := range links {
		// チーム名は代表が書いたものなので「」で囲む(MemberLinkText と同じ)
		text := fmt.Sprintf("「%s」%sの記録係です。\n", link.Team, link.Tournament)
		if len(link.Matches) == 0 {
			text += "これから記録する試合はまだありません。時間割が決まると、ここに試合が出ます。"
		} else {
			text += "試合を選ぶと、その試合のスコアの画面が開きます。"
		}
		choices := make([]Choice, 0, len(link.Matches)+1)
		for _, match := range link.Matches {
			choices = append(choices, Choice{Label: clip(MatchLabel(match), MaxChoiceLabel),
				URL: fmt.Sprintf("%s?match=%d", link.URL, match.ID)})
		}
		choices = append(choices, Choice{Label: "自分たちの試合の一覧", URL: link.URL})
		out[i] = Message{Text: text + "\n\n" + linkNote(), Choices: choices}
	}
	return out
}

// MatchLabel は試合のボタンの文字(「2/9 10:00 コート 3 対 神戸」。分からないところは書かない)。
func MatchLabel(match entry.ScorerMatch) string {
	parts := []string{}
	if match.Date != nil {
		parts = append(parts, fmt.Sprintf("%d/%d", int(match.Date.Month()), match.Date.Day()))
	}
	if match.Start != "" {
		parts = append(parts, match.Start)
	}
	if match.Field != "" {
		parts = append(parts, "コート "+match.Field)
	}
	opponent := "相手未定"
	if match.Opponent != "" {
		opponent = "対 " + match.Opponent
	}
	return strings.Join(append(parts, opponent), " ")
}

// MemberLinkMessage はメンバー登録のリンクの返信(招待リンクから「参加」を送った人へ)。チーム名は代表が
// 書いたものなので「」で囲み、公式アカウントの文と見分けられるようにする(改行は entry が拒む)。
func MemberLinkMessage(team, url string) Message {
	return Message{Text: fmt.Sprintf("「%s」のメンバー登録です。\n下のボタンから、自分の名前を入れて登録してください。\n\n%s",
		team, linkNote()), Choices: []Choice{{Label: "登録の画面を開く", URL: url}}}
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
