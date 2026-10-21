package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// apiTimeout は 1 リクエストのタイムアウト。手元の api への問い合わせなので短くてよい。
const apiTimeout = 30 * time.Second

// apiClient は閲覧用エンドポイントを叩く。応答は docs/api.md の形のうち、確かめる値だけを読む。
type apiClient struct {
	base  string
	token string
	http  *http.Client
}

func newAPIClient(base, token string) *apiClient {
	return &apiClient{base: base, token: token, http: &http.Client{Timeout: apiTimeout}}
}

// get は path を GET して out に読む。HTTP ステータスと、通信や JSON の失敗を返す。
// 200 以外のときは out に読まない。
func (c *apiClient) get(ctx context.Context, path string, out any) (int, error) {
	// URL は -api フラグ(手元で起動した api)とこのコマンド内で組み立てたパスだけからなる
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil) //nolint:gosec // 上記
	if err != nil {
		return 0, fmt.Errorf("new request %s: %w", path, err)
	}
	req.Header.Set("X-Internal-Token", c.token)
	resp, err := c.http.Do(req) //nolint:gosec // G704: 同上
	if err != nil {
		return 0, fmt.Errorf("get %s: %w", path, err)
	}
	defer func() {
		_ = resp.Body.Close() // 本文は読み終えている。Close の失敗は結果に影響しない
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("read %s: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return resp.StatusCode, fmt.Errorf("decode %s: %w", path, err)
	}
	return resp.StatusCode, nil
}

type i18nDTO struct {
	Ja string `json:"ja"`
	En string `json:"en"`
}

type teamRefDTO struct {
	ID   int64   `json:"id"`
	Name i18nDTO `json:"name"`
}

type listDTO struct {
	Sections []struct {
		Status      string `json:"status"`
		Tournaments []struct {
			ID int64 `json:"id"`
		} `json:"tournaments"`
	} `json:"sections"`
}

type frameDTO struct {
	ID        int64   `json:"id"`
	Name      i18nDTO `json:"name"`
	ShortName string  `json:"shortName"`
	Divisions []struct {
		ID     int64   `json:"id"`
		Name   i18nDTO `json:"name"`
		Groups []struct {
			ID     int64  `json:"id"`
			Name   string `json:"name"`
			Linked bool   `json:"linked"`
		} `json:"groups"`
	} `json:"divisions"`
	Galleries []struct {
		ID int64 `json:"id"`
	} `json:"galleries"`
}

type topDTO struct {
	Frame frameDTO `json:"frame"`
}

type teamsDTO struct {
	Divisions []struct {
		ID    int64 `json:"id"`
		Teams []struct {
			ID   int64   `json:"id"`
			Name i18nDTO `json:"name"`
		} `json:"teams"`
	} `json:"divisions"`
}

type matchSideDTO struct {
	Team        *teamRefDTO `json:"team"`
	Placeholder *struct {
		Code    string `json:"code"`
		Outcome string `json:"outcome"`
	} `json:"placeholder"`
	Score *int `json:"score"`
}

type matchRowDTO struct {
	MatchID int64        `json:"matchId"`
	Code    string       `json:"code"`
	Date    *string      `json:"date"`
	Status  string       `json:"status"`
	Left    matchSideDTO `json:"left"`
	Right   matchSideDTO `json:"right"`
}

type scheduleDTO struct {
	Rows []matchRowDTO `json:"rows"`
}

type standingsDTO struct {
	Standings []struct {
		Team teamRefDTO `json:"team"`
		Rank *int       `json:"rank"`
	} `json:"standings"`
}

type groupDTO struct {
	Group struct {
		Kind string `json:"kind"`
	} `json:"group"`
	Pool *struct {
		Results []struct {
			Team teamRefDTO `json:"team"`
			Rank *int       `json:"rank"`
		} `json:"results"`
	} `json:"pool"`
	Bracket *struct {
		Rows [][]struct {
			MatchID *int64 `json:"matchId"`
		} `json:"rows"`
	} `json:"bracket"`
	Matches []matchRowDTO `json:"matches"`
}

type teamDTO struct {
	Players []struct {
		Player struct {
			ID int64 `json:"id"`
		} `json:"player"`
	} `json:"players"`
	Staff []struct {
		ID int64 `json:"id"`
	} `json:"staff"`
}

type matchDTO struct {
	Match struct {
		Code string `json:"code"`
		Left struct {
			Score *int `json:"score"`
		} `json:"left"`
		Right struct {
			Score *int `json:"score"`
		} `json:"right"`
	} `json:"match"`
	Events []struct {
		LeftScore  int `json:"leftScore"`
		RightScore int `json:"rightScore"`
	} `json:"events"`
}

// ignoredDTO は 200 で返るかだけを確かめるエンドポイントの応答。
type ignoredDTO struct{}

func (c *apiClient) tournamentList(ctx context.Context) (*listDTO, error) {
	var out listDTO
	status, err := c.get(ctx, "/api/v1/tournaments", &out)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/v1/tournaments: HTTP %d", status)
	}
	return &out, nil
}
