// Package auth はパスワードの照合、セッショントークン、ログインの手順を持つ。
// 行の読み書きは store に任せ、ここは秘密の扱い(ハッシュ、乱数、定数時間比較)と手順だけを持つ。
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"

	"scoreview/api/internal/domain"
)

// argon2id のパラメータ。OWASP Password Storage Cheat Sheet の argon2id の推奨構成の 1 つ
// 「m=19456 (19 MiB), t=2, p=1」に合わせる
// (https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#argon2id)。
// 塩 16 バイトと鍵 32 バイトは RFC 9106 の推奨(4 章: salt 128 bit、tag 256 bit)。
// 値を変えても既存のハッシュは PHC 文字列に書かれたパラメータで照合できる。
const (
	argonMemoryKiB uint32 = 19 * 1024
	argonTime      uint32 = 2
	argonThreads   uint8  = 1
	argonKeyLen    uint32 = 32
	argonSaltLen          = 16
)

// パスワードの長さ(文字数)。下限は OWASP ASVS 4.0.3 V2.1.1(12 文字以上)。上限は、argon2 自体は
// 長さを問わないが、巨大な入力で CPU を占有されないための値(ASVS V2.1.2 の 128 文字以上を許す)。
const (
	MinPasswordLen = 12
	MaxPasswordLen = 256
)

// dummyHash は存在しないメールアドレスでのログインでも照合する PHC 文字列。照合の処理時間を
// 実在するユーザーと揃え、応答時間からアドレスの有無を推測されないようにする。上の定数と同じ
// パラメータで作った値で、パラメータを変えたら作り直す(TestDummyHashUsesCurrentParams が検査する)。
// 元のパスワードは捨てた乱数で、どの入力とも一致しない。
const dummyHash = "$argon2id$v=19$m=19456,t=2,p=1$" +
	"hBJRBPonq2i2RKVxcnxoXQ$ouMRbVdzNJatCPaL0ZsibVnSJp6gtBnGZi3QJ3yA4zw"

// ErrMalformedHash は保存されたハッシュが argon2id の PHC 文字列として読めないことを表す。
var ErrMalformedHash = errors.New("malformed password hash")

// ValidatePassword はパスワードの長さを検査し、同じ文字の繰り返しとよく使われるパスワードを拒否する
// (checkPasswordStrength)。
func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < MinPasswordLen || length > MaxPasswordLen {
		return domain.Invalid(
			"password",
			fmt.Sprintf("must be %d to %d characters", MinPasswordLen, MaxPasswordLen),
		)
	}
	return checkPasswordStrength(password)
}

// HashPassword はパスワードを argon2id でハッシュし、PHC 文字列
// ($argon2id$v=19$m=...,t=...,p=...$salt$hash)で返す。
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}
	key := argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemoryKiB,
		argonThreads,
		argonKeyLen,
	)
	return encodePHC(
		params{memory: argonMemoryKiB, time: argonTime, threads: argonThreads},
		salt,
		key,
	), nil
}

// VerifyPassword は password が PHC 文字列 encoded と一致するかを定数時間で比べる。
// 照合はハッシュに書かれたパラメータで行う。
func VerifyPassword(encoded, password string) (bool, error) {
	p, salt, want, err := decodePHC(encoded)
	if err != nil {
		return false, err
	}
	keyLen := uint32(len(want)) //nolint:gosec // len(want) は decodePHC が maxKeyLen 以下に限っている
	got := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, keyLen)
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// verifyDummy は存在しないユーザーのときに、実在するユーザーと同じ量の計算をする。
func verifyDummy(password string) {
	// 結果は常に不一致で、使わない。処理時間を揃えることだけが目的
	_, _ = VerifyPassword(dummyHash, password)
}

type params struct {
	memory  uint32
	time    uint32
	threads uint8
}

func encodePHC(p params, salt, key []byte) string {
	enc := base64.RawStdEncoding
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.memory,
		p.time,
		p.threads,
		enc.EncodeToString(salt),
		enc.EncodeToString(key),
	)
}

// maxKeyLen は照合する鍵の長さの上限。壊れたハッシュで巨大な計算をしないため。
const maxKeyLen = 64

func decodePHC(encoded string) (params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// "", "argon2id", "v=19", "m=..,t=..,p=..", salt, hash
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return params{}, nil, nil, ErrMalformedHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return params{}, nil, nil, fmt.Errorf("%w: version %q", ErrMalformedHash, parts[2])
	}
	var p params
	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&p.memory,
		&p.time,
		&p.threads,
	); err != nil {
		return params{}, nil, nil, fmt.Errorf("%w: params %q", ErrMalformedHash, parts[3])
	}
	if p.memory == 0 || p.time == 0 || p.threads == 0 {
		return params{}, nil, nil, fmt.Errorf("%w: zero parameter", ErrMalformedHash)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params{}, nil, nil, fmt.Errorf("%w: salt: %w", ErrMalformedHash, err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) == 0 || len(key) > maxKeyLen {
		return params{}, nil, nil, fmt.Errorf("%w: key", ErrMalformedHash)
	}
	return p, salt, key, nil
}
