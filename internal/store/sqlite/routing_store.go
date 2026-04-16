package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"opencrab/internal/domain"
)

const routingStrategySettingKey = "gateway.routing_strategy"
const gatewayCooldownSecondsKey = "gateway.cooldown_seconds"
const gatewayStickyEnabledKey = "gateway.sticky_enabled"
const gatewayStickyKeySourceKey = "gateway.sticky_key_source"

type RoutingConfigStore struct {
	db *sql.DB
}

type RoutingCursorStore struct {
	db *sql.DB
}

type GatewayRuntimeConfigStore struct {
	db *sql.DB
}

type RoutingRuntimeStateStore struct {
	db *sql.DB
}

type StickyRoutingStore struct {
	db *sql.DB
}

func NewRoutingConfigStore(db *sql.DB) *RoutingConfigStore {
	return &RoutingConfigStore{db: db}
}

func NewRoutingCursorStore(db *sql.DB) *RoutingCursorStore {
	return &RoutingCursorStore{db: db}
}

func NewGatewayRuntimeConfigStore(db *sql.DB) *GatewayRuntimeConfigStore {
	return &GatewayRuntimeConfigStore{db: db}
}

func NewRoutingRuntimeStateStore(db *sql.DB) *RoutingRuntimeStateStore {
	return &RoutingRuntimeStateStore{db: db}
}

func NewStickyRoutingStore(db *sql.DB) *StickyRoutingStore {
	return &StickyRoutingStore{db: db}
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
	currentIndex, err := s.GetRoutingCursor(ctx, routeKey)
	if err != nil {
		return err
	}
	if currentIndex == nextIndex {
		return nil
	}
	_, err = s.db.ExecContext(
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

func (s *GatewayRuntimeConfigStore) GetGatewayRuntimeSettings(ctx context.Context) (domain.GatewayRuntimeSettings, error) {
	settings := domain.GatewayRuntimeSettings{
		CooldownDuration: 45 * time.Second,
		StickyEnabled:    true,
		StickyKeySource:  "auto",
	}

	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM system_settings WHERE key IN (?, ?, ?)`, gatewayCooldownSecondsKey, gatewayStickyEnabledKey, gatewayStickyKeySourceKey)
	if err != nil {
		return settings, fmt.Errorf("读取网关运行时设置失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return settings, fmt.Errorf("读取网关运行时设置项失败: %w", err)
		}
		switch key {
		case gatewayCooldownSecondsKey:
			seconds, convErr := strconv.Atoi(strings.TrimSpace(value))
			if convErr == nil && seconds > 0 {
				settings.CooldownDuration = time.Duration(seconds) * time.Second
			}
		case gatewayStickyEnabledKey:
			normalized := strings.ToLower(strings.TrimSpace(value))
			settings.StickyEnabled = normalized != "false" && normalized != "0" && normalized != "禁用"
		case gatewayStickyKeySourceKey:
			normalized := strings.ToLower(strings.TrimSpace(value))
			if normalized == "header" || normalized == "metadata" || normalized == "auto" {
				settings.StickyKeySource = normalized
			}
		}
	}
	if err := rows.Err(); err != nil {
		return settings, fmt.Errorf("遍历网关运行时设置失败: %w", err)
	}

	return settings, nil
}

func (s *RoutingRuntimeStateStore) MarkCooldown(ctx context.Context, routeID int64, duration time.Duration, lastError string) (string, error) {
	if routeID <= 0 {
		return "", nil
	}
	cooldownUntil := time.Now().Add(duration).Format(time.RFC3339)
	updatedAt := time.Now().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `INSERT INTO routing_runtime_states(route_id, cooldown_until, last_error, updated_at) VALUES (?, ?, ?, ?) ON CONFLICT(route_id) DO UPDATE SET cooldown_until = excluded.cooldown_until, last_error = excluded.last_error, updated_at = excluded.updated_at`, routeID, cooldownUntil, lastError, updatedAt)
	if err != nil {
		return "", fmt.Errorf("写入 cooldown 状态失败: %w", err)
	}
	return cooldownUntil, nil
}

func (s *RoutingRuntimeStateStore) ClearCooldown(ctx context.Context, routeID int64) error {
	if routeID <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `UPDATE routing_runtime_states SET cooldown_until = '', last_error = '', updated_at = ? WHERE route_id = ? AND (cooldown_until <> '' OR last_error <> '')`, time.Now().Format(time.RFC3339), routeID)
	if err != nil {
		return fmt.Errorf("清理 cooldown 状态失败: %w", err)
	}
	return nil
}

func (s *RoutingRuntimeStateStore) CountActiveCooldowns(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM routing_runtime_states WHERE cooldown_until <> '' AND datetime(cooldown_until) > datetime('now')`).Scan(&count); err != nil {
		return 0, fmt.Errorf("统计 active cooldown 失败: %w", err)
	}
	return count, nil
}

func (s *StickyRoutingStore) GetStickyBinding(ctx context.Context, affinityKey string, modelAlias string, protocol domain.Protocol) (int64, bool, error) {
	var routeID int64
	err := s.db.QueryRowContext(ctx, `SELECT route_id FROM routing_affinity_bindings WHERE affinity_key = ? AND model_alias = ? AND protocol = ? LIMIT 1`, affinityKey, modelAlias, string(protocol)).Scan(&routeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("读取 sticky 绑定失败: %w", err)
	}
	return routeID, true, nil
}

func (s *StickyRoutingStore) UpsertStickyBinding(ctx context.Context, affinityKey string, modelAlias string, protocol domain.Protocol, routeID int64) error {
	if strings.TrimSpace(affinityKey) == "" || routeID <= 0 {
		return nil
	}
	currentRouteID, found, err := s.GetStickyBinding(ctx, affinityKey, modelAlias, protocol)
	if err != nil {
		return err
	}
	if found && currentRouteID == routeID {
		return nil
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO routing_affinity_bindings(affinity_key, model_alias, protocol, route_id, updated_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(affinity_key, model_alias, protocol) DO UPDATE SET route_id = excluded.route_id, updated_at = excluded.updated_at`, affinityKey, modelAlias, string(protocol), routeID, time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("写入 sticky 绑定失败: %w", err)
	}
	return nil
}

func (s *StickyRoutingStore) CountStickyBindings(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM routing_affinity_bindings`).Scan(&count); err != nil {
		return 0, fmt.Errorf("统计 sticky 绑定失败: %w", err)
	}
	return count, nil
}
