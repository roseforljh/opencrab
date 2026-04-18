package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"opencrab/internal/capability"
	"opencrab/internal/domain"
)

type CapabilityProfileStore struct {
	db *sql.DB
}

type capabilityProfileConfig struct {
	Enabled      *bool    `json:"enabled,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

func NewCapabilityProfileStore(db *sql.DB) *CapabilityProfileStore {
	return &CapabilityProfileStore{db: db}
}

func (s *CapabilityProfileStore) ListCapabilityProfiles(ctx context.Context) ([]capability.ProfileRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT scope_type, scope_key, operation, config_json
FROM capability_profiles
ORDER BY CASE scope_type
	WHEN 'provider_default' THEN 1
	WHEN 'channel_override' THEN 2
	WHEN 'model_profile' THEN 3
	ELSE 9
END, scope_key, operation`)
	if err != nil {
		return nil, fmt.Errorf("查询 capability_profiles 失败: %w", err)
	}
	defer rows.Close()

	records := make([]capability.ProfileRecord, 0)
	for rows.Next() {
		var scopeType string
		var scopeKey string
		var operation string
		var configJSON string
		if err := rows.Scan(&scopeType, &scopeKey, &operation, &configJSON); err != nil {
			return nil, fmt.Errorf("读取 capability_profiles 失败: %w", err)
		}

		record := capability.ProfileRecord{
			ScopeType: capability.ScopeType(strings.TrimSpace(scopeType)),
			ScopeKey:  strings.TrimSpace(scopeKey),
			Operation: domain.ProtocolOperation(strings.TrimSpace(operation)),
		}
		if strings.TrimSpace(configJSON) != "" {
			var config capabilityProfileConfig
			if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
				return nil, fmt.Errorf("解析 capability_profiles 配置失败: %w", err)
			}
			record.Enabled = config.Enabled
			if config.Capabilities != nil {
				record.Capabilities = make([]capability.Capability, 0, len(config.Capabilities))
				for _, item := range config.Capabilities {
					record.Capabilities = append(record.Capabilities, capability.Capability(strings.TrimSpace(item)))
				}
			}
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 capability_profiles 失败: %w", err)
	}
	return records, nil
}

func (s *CapabilityProfileStore) UpsertCapabilityProfile(ctx context.Context, input domain.UpsertCapabilityProfileInput) error {
	config := capabilityProfileConfig{
		Enabled:      input.Enabled,
		Capabilities: normalizeCapabilityList(input.Capabilities),
	}
	payload, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("编码 capability profile 失败: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO capability_profiles(scope_type, scope_key, operation, config_json, updated_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(scope_type, scope_key, operation) DO UPDATE SET config_json = excluded.config_json, updated_at = excluded.updated_at`,
		strings.TrimSpace(input.ScopeType),
		strings.TrimSpace(input.ScopeKey),
		strings.TrimSpace(input.Operation),
		string(payload),
		time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("写入 capability profile 失败: %w", err)
	}
	return nil
}

func (s *CapabilityProfileStore) DeleteCapabilityProfile(ctx context.Context, input domain.DeleteCapabilityProfileInput) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM capability_profiles WHERE scope_type = ? AND scope_key = ? AND operation = ?`,
		strings.TrimSpace(input.ScopeType),
		strings.TrimSpace(input.ScopeKey),
		strings.TrimSpace(input.Operation),
	)
	if err != nil {
		return fmt.Errorf("删除 capability profile 失败: %w", err)
	}
	return nil
}

func normalizeCapabilityList(items []string) []string {
	if items == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}
