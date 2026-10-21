package domain

// 管理画面と管理 API で作る資源の英語名の補い(通し確認で、名簿の貼り付けが日本語名しか入れず、英語の
// 閲覧ページでチーム名が空になった)。閲覧ページは英語が空なら空で出す(参照サイトの再現)ので、表示では
// 補わず、作るときにデータの側で日本語の値を入れる。取込ツールのデータは参照サイトの値をそのまま持つので
// 対象にしない。

// WithEnglish は En が空なら Ja を入れた値を返す。En があればそのまま返す。
func (t I18nText) WithEnglish() I18nText {
	if t.En == "" {
		t.En = t.Ja
	}
	return t
}

// TeamNamesWithEnglish はチームの正式名と短縮名の英語を補う。短縮名の英語が空なら、英語の正式名が
// 指定されていればそれを、無ければ日本語の短縮名を入れる。英語の正式名が空なら日本語の正式名を入れる。
func TeamNamesWithEnglish(full, short I18nText) (I18nText, I18nText) {
	if short.En == "" && full.En != "" {
		short.En = full.En
	}
	return full.WithEnglish(), short.WithEnglish()
}
