package auth

import (
	_ "embed" // commonPasswordList を埋め込む
	"strings"

	"scoreview/api/internal/domain"
)

// commonPasswordList はよく使われるパスワードのうち、長さの下限(MinPasswordLen)以上のものを小文字に
// して 1 行に 1 つ並べた一覧。NCSC(英国国家サイバーセキュリティセンター)が Have I Been Pwned の
// 漏洩データから作った上位 10 万件(SecLists の Passwords/Common-Credentials/
// 100k-most-used-passwords-NCSC.txt)から、12 文字以上を取り出した 1,193 件。外部のサービスには
// 問い合わせない(ログインの度に第三者へパスワードの手がかりを送らないため)。
//
//go:embed common_passwords.txt
var commonPasswordList string

// maxRepeatUnit は「同じ並びの繰り返し」とみなす並びの長さの上限。"aaaaaaaaaaaa"(1 文字)、
// "abababababab"(2 文字)、"abcabcabcabc"、"12341234123412341"(4 文字)を拒否する。
const maxRepeatUnit = 4

// checkPasswordStrength は、同じ文字(または短い並び)の繰り返しと、よく使われるパスワードを拒否する
// (OWASP ASVS 4.0.3 V2.1.7: 漏洩したパスワードの一覧と照合する。セキュリティ点検 L-8)。
// 大文字と小文字は区別しない("Password1234" も "password1234" と同じく拒否する)。
func checkPasswordStrength(password string) error {
	if isRepetition([]rune(password)) {
		return domain.Invalid("password", "must not be a repetition of the same few characters")
	}
	if isCommonPassword(password) {
		return domain.Invalid("password", "is too common; choose a password others are unlikely to use")
	}
	return nil
}

// isRepetition は runes が maxRepeatUnit 文字以下の並びの繰り返しかどうか。
func isRepetition(runes []rune) bool {
	for unit := 1; unit <= maxRepeatUnit && unit < len(runes); unit++ {
		repeated := true
		for i := unit; i < len(runes); i++ {
			if runes[i] != runes[i%unit] {
				repeated = false
				break
			}
		}
		if repeated {
			return true
		}
	}
	return false
}

// isCommonPassword は password が commonPasswordList にあるかどうか(大文字と小文字は区別しない)。
func isCommonPassword(password string) bool {
	return strings.Contains("\n"+commonPasswordList, "\n"+strings.ToLower(password)+"\n")
}
