package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// BracketCellKind はトーナメント表のセルの種類。値は参照サイトの td のクラス名(clone-spec C 2.3)。
// 空文字は罫線だけのセル。
type BracketCellKind string

// BracketCellKind の値。
const (
	BracketEmpty    BracketCellKind = ""
	BracketTeam     BracketCellKind = "team"
	BracketWin      BracketCellKind = "win"
	BracketLose     BracketCellKind = "lose"
	BracketGame     BracketCellKind = "game"
	BracketTxtYen   BracketCellKind = "txt_yen"
	BracketTxtSharp BracketCellKind = "txt_sharp"
	BracketTxtAnd   BracketCellKind = "txt_and"
	// 以下 3 つは参照サイトの旧い大会の格子に出る(rank は 33 セルで観測。txt_left/right は CSS 定義のみ)
	BracketRank     BracketCellKind = "rank"
	BracketTxtLeft  BracketCellKind = "txt_left"
	BracketTxtRight BracketCellKind = "txt_right"
)

func (k BracketCellKind) valid() bool {
	switch k {
	case BracketEmpty, BracketTeam, BracketWin, BracketLose, BracketGame,
		BracketTxtYen, BracketTxtSharp, BracketTxtAnd, BracketRank, BracketTxtLeft, BracketTxtRight:
		return true
	}
	return false
}

// BracketBorders はセルの罫線(BdTop / BdBtm / BdLeft / BdRight)。
type BracketBorders struct {
	Top    bool
	Bottom bool
	Left   bool
	Right  bool
}

// BracketCell はトーナメント表の 1 セル。TeamID はチーム名のセル、MatchID は game セルが参照する。
type BracketCell struct {
	Borders BracketBorders
	Kind    BracketCellKind
	Text    string
	TeamID  *int64
	MatchID *int64
}

// BracketLayout はトーナメント表の格子。参照サイトの表は 1 枚の table の罫線だけで描かれ、
// テンプレートごとに形が違うので、形を計算せず格子のまま保存する(docs/architecture.md)。
// 全行が Columns 個のセルを持つ(参照サイトの td は全て同じ WDiv{N} で colspan が無い)。
type BracketLayout struct {
	Columns int
	Rows    [][]BracketCell
}

// ErrInvalidBracketLayout は groups.bracket_layout の内容が形式に合わないことを表す。
var ErrInvalidBracketLayout = errors.New("invalid bracket layout")

// Validate は格子の形とセルの種類を検査する。
func (l BracketLayout) Validate() error {
	if l.Columns <= 0 {
		return fmt.Errorf("%w: columns must be positive, got %d", ErrInvalidBracketLayout, l.Columns)
	}
	for r, row := range l.Rows {
		if len(row) != l.Columns {
			return fmt.Errorf("%w: row %d has %d cells, want %d",
				ErrInvalidBracketLayout, r, len(row), l.Columns)
		}
		for c, cell := range row {
			if !cell.Kind.valid() {
				return fmt.Errorf("%w: row %d col %d has unknown kind %q",
					ErrInvalidBracketLayout, r, c, cell.Kind)
			}
		}
	}
	return nil
}

// 以下は groups.bracket_layout(JSONB)の保存形式。HTTP の DTO ではない。
// 00001_init.sql がこのファイルを形式の定義として参照しているので domain に置く。
// domain の型そのものには JSON タグを付けず、保存形式の構造体を分けて持つ。

type bracketLayoutJSON struct {
	Columns int                 `json:"columns"`
	Rows    [][]bracketCellJSON `json:"rows"`
}

type bracketCellJSON struct {
	Borders bracketBordersJSON `json:"borders"`
	Kind    string             `json:"kind"`
	Text    string             `json:"text"`
	TeamID  *int64             `json:"teamId"`
	MatchID *int64             `json:"matchId"`
}

type bracketBordersJSON struct {
	Top    bool `json:"top"`
	Bottom bool `json:"bottom"`
	Left   bool `json:"left"`
	Right  bool `json:"right"`
}

// MarshalBracketLayout は格子を保存形式の JSON にする。形式に合わない格子はエラーにする。
func MarshalBracketLayout(l BracketLayout) ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	wire := bracketLayoutJSON{Columns: l.Columns, Rows: make([][]bracketCellJSON, len(l.Rows))}
	for r, row := range l.Rows {
		wire.Rows[r] = make([]bracketCellJSON, len(row))
		for c, cell := range row {
			wire.Rows[r][c] = bracketCellJSON{
				Borders: bracketBordersJSON(cell.Borders),
				Kind:    string(cell.Kind),
				Text:    cell.Text,
				TeamID:  cell.TeamID,
				MatchID: cell.MatchID,
			}
		}
	}
	data, err := json.Marshal(wire)
	if err != nil {
		return nil, fmt.Errorf("marshal bracket layout: %w", err)
	}
	return data, nil
}

// ParseBracketLayout は保存形式の JSON を読み、形と種類を検査して返す。未知のキーはエラーにする
// (書き手と読み手の食い違いを黙って捨てないため)。
func ParseBracketLayout(data []byte) (BracketLayout, error) {
	var wire bracketLayoutJSON
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&wire); err != nil {
		return BracketLayout{}, fmt.Errorf("%w: %w", ErrInvalidBracketLayout, err)
	}
	layout := BracketLayout{Columns: wire.Columns, Rows: make([][]BracketCell, len(wire.Rows))}
	for r, row := range wire.Rows {
		layout.Rows[r] = make([]BracketCell, len(row))
		for c, cell := range row {
			layout.Rows[r][c] = BracketCell{
				Borders: BracketBorders(cell.Borders),
				Kind:    BracketCellKind(cell.Kind),
				Text:    cell.Text,
				TeamID:  cell.TeamID,
				MatchID: cell.MatchID,
			}
		}
	}
	if err := layout.Validate(); err != nil {
		return BracketLayout{}, err
	}
	return layout, nil
}
