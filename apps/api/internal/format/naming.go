package format

import (
	"fmt"
	"strings"

	"scoreview/api/internal/domain"
)

// groupLetter は開始文字 start から offset 個進めたグループ文字を返す。Z の次は AA、AB と
// 表計算の列名と同じ規則で続ける(参照サイトは T までしか観測しておらず、Z を超える規則は推測)。
func groupLetter(start string, offset int) (string, error) {
	base, err := letterIndex(start)
	if err != nil {
		return "", err
	}
	return letterName(base + offset), nil
}

// letterIndex は "A"=0、"Z"=25、"AA"=26 の添字を返す。
func letterIndex(letters string) (int, error) {
	if letters == "" {
		return 0, domain.Invalid("startLetter", "empty")
	}
	index := 0
	for _, r := range letters {
		if r < 'A' || r > 'Z' {
			return 0, domain.Invalid("startLetter", fmt.Sprintf("%q is not A-Z", letters))
		}
		index = index*26 + int(r-'A') + 1
	}
	return index - 1, nil
}

func letterName(index int) string {
	var sb strings.Builder
	var rev []byte
	for n := index + 1; n > 0; n = (n - 1) / 26 {
		rev = append(rev, byte('A'+(n-1)%26))
	}
	for i := len(rev) - 1; i >= 0; i-- {
		sb.WriteByte(rev[i])
	}
	return sb.String()
}

// ordinal は英語の序数("1st"、"22nd")。参照サイトの "21th"(clone-spec C 2.3)の誤りは、
// 取込データの表示でだけ再現し、エンジンが新しく作る名前では正しい綴りにする。
func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}
