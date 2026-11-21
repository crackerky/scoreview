package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// TokenBytes は推測できないトークン(ログインのセッション、本人用のリンクの合言葉)の乱数のバイト数。
// 256 bit は総当たりで当てられない長さ。
const TokenBytes = 32

// NewToken は random から推測できないトークンを作る。URL にそのまま入れられる 43 文字(base64url、
// パディング無し)。本番の random は crypto/rand。
func NewToken(random io.Reader) (string, error) {
	raw := make([]byte, TokenBytes)
	if _, err := io.ReadFull(random, raw); err != nil {
		return "", fmt.Errorf("read random bytes for token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// HashToken はトークンを保存と照合に使う SHA-256 にする。トークンは 256 bit の乱数なので、
// 塩も遅いハッシュも要らない(辞書攻撃が成り立たない)。
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
