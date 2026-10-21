package format

import (
	"bytes"
	"encoding/json"
	"fmt"

	"scoreview/api/internal/domain"
)

// Spec.Params の JSON 形(division_formats.params と管理 API の params。docs/format-engine.md
// 「形の指定の JSON」)。キーは camelCase で、テンプレートごとに次の形を取る。
//
//	pool_bands:  {"teams":12,"pools":4,"bands":[{"ranks":[1],"shape":"T4","pairing":null,"name":null}]}
//	blocks:      {"teams":32,"bands":[{"ranks":[2,3],"shape":"T8R1","pairing":null,"name":null}]}
//	round_robin: {"teams":8,"pools":2,"finalPoolRanks":[1,2]}
//
// pairing と name は null で既定(Band のコメント)。
type paramsJSON struct {
	Teams          int        `json:"teams"`
	Pools          *int       `json:"pools,omitempty"`
	Bands          []bandJSON `json:"bands,omitempty"`
	FinalPoolRanks []int      `json:"finalPoolRanks,omitempty"`
}

type bandJSON struct {
	Ranks   []int    `json:"ranks"`
	Shape   Shape    `json:"shape"`
	Pairing *Pairing `json:"pairing"`
	Name    *string  `json:"name"`
}

// DecodeParams は template の Params を JSON から読む。戻り値は PoolBandsParams、BlocksParams、
// RoundRobinParams のどれか(Engine.Generate にそのまま渡せる)。未知のキーや、テンプレートに
// 無いキーは domain.ValidationError。
func DecodeParams(template string, data []byte) (any, error) {
	var raw paramsJSON
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return nil, domain.Invalid("params", err.Error())
	}
	bands := make([]Band, len(raw.Bands))
	for i, b := range raw.Bands {
		bands[i] = Band{Ranks: b.Ranks, Shape: b.Shape}
		if b.Pairing != nil {
			bands[i].Pairing = *b.Pairing
		}
		if b.Name != nil {
			bands[i].Name = *b.Name
		}
	}
	pools := 0
	if raw.Pools != nil {
		pools = *raw.Pools
	}
	switch template {
	case TemplatePoolBands:
		if len(raw.FinalPoolRanks) > 0 {
			return nil, domain.Invalid("params.finalPoolRanks", "is only for round_robin")
		}
		return PoolBandsParams{Teams: raw.Teams, Pools: pools, Bands: bands}, nil
	case TemplateBlocks:
		if raw.Pools != nil || len(raw.FinalPoolRanks) > 0 {
			return nil, domain.Invalid("params", "blocks takes teams and bands only")
		}
		return BlocksParams{Teams: raw.Teams, Bands: bands}, nil
	case TemplateRoundRobin:
		if len(raw.Bands) > 0 {
			return nil, domain.Invalid("params.bands", "is not for round_robin")
		}
		return RoundRobinParams{Teams: raw.Teams, Pools: pools, FinalPoolRanks: raw.FinalPoolRanks}, nil
	}
	return nil, domain.Invalid("template", fmt.Sprintf("unknown template %q", template))
}

// EncodeParams は DecodeParams で読める JSON にする。params はテンプレートのパラメータの型
// (値かポインタ)でなければエラー。
func EncodeParams(params any) ([]byte, error) {
	var raw paramsJSON
	switch p := params.(type) {
	case PoolBandsParams:
		raw = paramsJSON{Teams: p.Teams, Pools: &p.Pools, Bands: bandsJSON(p.Bands)}
	case *PoolBandsParams:
		return EncodeParams(*p)
	case BlocksParams:
		raw = paramsJSON{Teams: p.Teams, Bands: bandsJSON(p.Bands)}
	case *BlocksParams:
		return EncodeParams(*p)
	case RoundRobinParams:
		raw = paramsJSON{Teams: p.Teams, Pools: &p.Pools, FinalPoolRanks: p.FinalPoolRanks}
	case *RoundRobinParams:
		return EncodeParams(*p)
	default:
		return nil, domain.Invalid("params", fmt.Sprintf("unsupported params %T", params))
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode params: %w", err)
	}
	return data, nil
}

// TeamsOf はパラメータのチーム数を返す。
func TeamsOf(params any) int {
	switch p := params.(type) {
	case PoolBandsParams:
		return p.Teams
	case BlocksParams:
		return p.Teams
	case RoundRobinParams:
		return p.Teams
	}
	return 0
}

func bandsJSON(bands []Band) []bandJSON {
	out := make([]bandJSON, len(bands))
	for i, b := range bands {
		out[i] = bandJSON{Ranks: b.Ranks, Shape: b.Shape}
		if b.Pairing != "" {
			pairing := b.Pairing
			out[i].Pairing = &pairing
		}
		if b.Name != "" {
			name := b.Name
			out[i].Name = &name
		}
	}
	return out
}
