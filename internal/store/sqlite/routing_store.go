package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"opencrab/internal/domain"
)

const routingStrategySettingKey = "gateway.routing_strategy"

type RoutingConfigStore struct {
	db *sql.DB
}

type RoutingCursorStore struct {
	db *sql.DB
}

func NewRoutingConfigStore(db *sql.DB) *RoutingConfigStore {
	return &RoutingConfigStore{db: db}
}

func NewRoutingCursorStore(db *sql.DB) *RoutingCursorStore {
	return &RoutingCursorStore{db: db}
}

func (s *RoutingConfigStore) GetRoutingStrategy(ctx context.Context) (domain.RoutingStrategy, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM system_settings WHERE key = ? LIMIT 1`, routingStrategySettingKey).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.RoutingStrategySequential, nil
		}
		return "", fmt.Errorf("读取路由策略失败: %w", err)
	}
	return domain.NormalizeRoutingStrategy(value), nil
}

func (s *RoutingCursorStore) GetRoutingCursor(ctx context.Context, routeKey string) (int, error) {
	var nextIndex int
	err := s.db.QueryRowContext(ctx, `SELECT next_index FROM routing_cursors WHERE route_key = ? LIMIT 1`, routeKey).Scan(&nextIndex)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("读取路由游标失败: %w", err)
	}
	if nextIndex < 0 {
		return 0, nil
	}
	return nextIndex, nil
}

func (s *RoutingCursorStore) AdvanceRoutingCursor(ctx context.Context, routeKey string, candidateCount int, selectedIndex int) error {
	if candidateCount <= 0 {
		return nil
	}
	nextIndex := (selectedIndex + 1) % candidateCount
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO routing_cursors(route_key, next_index, updated_at) VALUES (?, ?, ?) ON CONFLICT(route_key) DO UPDATE SET next_index = excluded.next_index, updated_at = excluded.updated_at`,
		routeKey,
		nextIndex,
		time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("写入路由游标失败: %w", err)
	}
	return nil
}
