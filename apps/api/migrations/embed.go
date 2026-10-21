// Package migrations はスキーマの正本である goose の SQL ファイルを埋め込む。
package migrations

import "embed"

// FS は migrations ディレクトリの *.sql。goose.SetBaseFS に渡す。
//
//go:embed *.sql
var FS embed.FS
