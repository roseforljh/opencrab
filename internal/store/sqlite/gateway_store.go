package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"opencrab/internal/domain"
)

type GatewayStore struct {
	db *sql.DB
}

func NewGatewayStore(db *sql.DB) *GatewayStore {
	return &GatewayStore{db: db}
}

func (s *GatewayStore) ListEnabledRoutesByModel(ctx context.Context, model string) ([]domain.GatewayRoute, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT mr.id, mr.model_alias, m.upstream_model, c.name, c.provider, c.endpoint, c.api_key, c.rpm_limit, c.max_inflight, c.safety_factor, c.dispatch_weight, mr.invocation_mode, mr.priority, mr.fallback_model, COALESCE(rrs.cooldown_until, ''), COALESCE(rrs.last_error, '')
FROM model_routes mr
JOIN models m ON m.alias = mr.model_alias
JOIN channels c ON c.name = mr.channel_name
LEFT JOIN routing_runtime_states rrs ON rrs.route_id = mr.id
WHERE mr.model_alias = ? AND c.enabled = 1
ORDER BY mr.priority ASC, mr.id ASC`, model)
	if err != nil {
		return nil, fmt.Errorf("查询执行路由失败: %w", err)
	}
	defer rows.Close()

	routes := make([]domain.GatewayRoute, 0)
	for rows.Next() {
		var route domain.GatewayRoute
		if err := rows.Scan(&route.ID, &route.ModelAlias, &route.UpstreamModel, &route.Channel.Name, &route.Channel.Provider, &route.Channel.Endpoint, &route.Channel.APIKey, &route.Channel.RPMLimit, &route.Channel.MaxInflight, &route.Channel.SafetyFactor, &route.Channel.DispatchWeight, &route.InvocationMode, &route.Priority, &route.FallbackModel, &route.CooldownUntil, &route.LastError); err != nil {
			return nil, fmt.Errorf("读取执行路由失败: %w", err)
		}
		routes = append(routes, route)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历执行路由失败: %w", err)
	}
	return routes, nil
}
