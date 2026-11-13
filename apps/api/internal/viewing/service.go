package viewing

import (
	"context"
	"fmt"
)

// Service は Source から記録を読み、掲示板を組み立てる Reader。
type Service struct {
	source Source
}

// NewService は source から読む Service を返す。
func NewService(source Source) *Service {
	return &Service{source: source}
}

// Board は Reader.Board。
func (s *Service) Board(ctx context.Context, tournamentID int64) (Board, error) {
	snapshot, err := s.source.Snapshot(ctx, tournamentID)
	if err != nil {
		return Board{}, fmt.Errorf("read board of tournament %d: %w", tournamentID, err)
	}
	return Assemble(snapshot), nil
}
