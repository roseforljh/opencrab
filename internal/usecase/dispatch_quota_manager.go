package usecase

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"opencrab/internal/domain"

	"github.com/redis/go-redis/v9"
)

var reserveDispatchQuotaScript = redis.NewScript(`
redis.call('ZREMRANGEBYSCORE', KEYS[2], '-inf', ARGV[1])
local nowMs = tonumber(ARGV[1])
local intervalMs = tonumber(ARGV[2])
local maxInflight = tonumber(ARGV[3])
local leaseMs = tonumber(ARGV[4])
local reservationKey = ARGV[5]
local cooldownUntilMs = tonumber(ARGV[6])
local healthPenaltyMs = tonumber(ARGV[7])
local stickyBiasMs = tonumber(ARGV[8])

local inflight = redis.call('ZCARD', KEYS[2])
local inflightReadyAt = nowMs
if maxInflight > 0 and inflight >= maxInflight then
  local firstLease = redis.call('ZRANGE', KEYS[2], 0, 0, 'WITHSCORES')
  if firstLease[2] then
    inflightReadyAt = tonumber(firstLease[2])
  end
end

local tatMs = tonumber(redis.call('HGET', KEYS[1], 'tat_ms') or '0')
if tatMs < nowMs then
  tatMs = nowMs
end

local dispatchAt = tatMs
if inflightReadyAt > dispatchAt then
  dispatchAt = inflightReadyAt
end
if cooldownUntilMs > dispatchAt then
  dispatchAt = cooldownUntilMs
end
dispatchAt = dispatchAt + healthPenaltyMs
if stickyBiasMs > 0 and dispatchAt > nowMs then
  dispatchAt = dispatchAt - stickyBiasMs
  if dispatchAt < nowMs then
    dispatchAt = nowMs
  end
end

local reservedTatMs = dispatchAt + intervalMs
redis.call('HSET', KEYS[1], 'tat_ms', reservedTatMs, 'updated_at_ms', nowMs)
local leaseAcquired = 0
if maxInflight > 0 and leaseMs > 0 and dispatchAt <= nowMs then
  redis.call('ZADD', KEYS[2], nowMs + leaseMs, reservationKey)
  leaseAcquired = 1
end

return {dispatchAt, dispatchAt - nowMs, inflight, reservedTatMs, inflightReadyAt, leaseAcquired}
`)

type RedisDispatchQuotaManager struct {
	settings  domain.DispatchRuntimeConfigStore
	mu        sync.Mutex
	client    *redis.Client
	signature string
}

func NewRedisDispatchQuotaManager(settings domain.DispatchRuntimeConfigStore) *RedisDispatchQuotaManager {
	return &RedisDispatchQuotaManager{settings: settings}
}

func (m *RedisDispatchQuotaManager) Reserve(ctx context.Context, input domain.DispatchReservationInput) (domain.DispatchReservationResult, error) {
	nowMs := time.Now().UnixMilli()
	result := domain.DispatchReservationResult{ChannelName: input.ChannelName, ReservationKey: normalizedReservationKey(input.ReservationKey), DispatchAtMs: nowMs, WaitMs: 0, CurrentInflight: 0, ReservedTatMs: nowMs, InflightReadyAtMs: nowMs, LeaseAcquired: true, Runtime: "disabled"}
	if m == nil || m.settings == nil {
		return result, nil
	}
	settings, err := m.settings.GetDispatchRuntimeSettings(ctx)
	if err != nil {
		result.Runtime = "config_error"
		return result, nil
	}
	if !settings.RedisEnabled {
		return result, nil
	}
	client, err := m.clientFor(ctx, settings)
	if err != nil {
		result.Runtime = "redis_unavailable"
		return result, nil
	}
	effectiveRPM := normalizeDispatchLimit(input.RPMLimit, input.SafetyFactor, settings.RetryReserveRatio)
	intervalMs := int64(60000 / effectiveRPM)
	if intervalMs <= 0 {
		intervalMs = 1
	}
	prefix := strings.TrimSpace(settings.RedisKeyPrefix)
	if prefix == "" {
		prefix = "opencrab"
	}
	channelKey := fmt.Sprintf("%s:quota:channel:%s", prefix, strings.TrimSpace(input.ChannelName))
	leaseKey := channelKey + ":leases"
	values, err := reserveDispatchQuotaScript.Run(ctx, client, []string{channelKey, leaseKey}, nowMs, intervalMs, max(0, input.MaxInflight), max64(0, input.LeaseMs), result.ReservationKey, max64(0, input.CooldownUntilMs), max64(0, input.HealthPenaltyMs), max64(0, input.StickyBiasMs)).Result()
	if err != nil {
		result.Runtime = "redis_error"
		return result, nil
	}
	items, ok := values.([]any)
	if !ok || len(items) != 6 {
		return domain.DispatchReservationResult{}, fmt.Errorf("redis quota reservation 返回值异常")
	}
	result.Runtime = "redis"
	result.DispatchAtMs = toInt64(items[0])
	result.WaitMs = toInt64(items[1])
	result.CurrentInflight = toInt64(items[2])
	result.ReservedTatMs = toInt64(items[3])
	result.InflightReadyAtMs = toInt64(items[4])
	result.LeaseAcquired = toInt64(items[5]) == 1
	return result, nil
}

func (m *RedisDispatchQuotaManager) Release(ctx context.Context, input domain.DispatchReleaseInput) error {
	if m == nil || m.settings == nil {
		return nil
	}
	settings, err := m.settings.GetDispatchRuntimeSettings(ctx)
	if err != nil || !settings.RedisEnabled {
		return nil
	}
	client, err := m.clientFor(ctx, settings)
	if err != nil {
		return nil
	}
	prefix := strings.TrimSpace(settings.RedisKeyPrefix)
	if prefix == "" {
		prefix = "opencrab"
	}
	leaseKey := fmt.Sprintf("%s:quota:channel:%s:leases", prefix, strings.TrimSpace(input.ChannelName))
	if err := client.ZRem(ctx, leaseKey, strings.TrimSpace(input.ReservationKey)).Err(); err != nil && err != redis.Nil {
		return fmt.Errorf("释放 dispatch lease 失败: %w", err)
	}
	return nil
}

func (m *RedisDispatchQuotaManager) clientFor(ctx context.Context, settings domain.DispatchRuntimeSettings) (*redis.Client, error) {
	signature := strings.Join([]string{settings.RedisAddress, strconv.Itoa(settings.RedisDB), settings.RedisPassword, strconv.FormatBool(settings.RedisTLSEnabled)}, "|")
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.client != nil && m.signature == signature {
		return m.client, nil
	}
	if m.client != nil {
		_ = m.client.Close()
	}
	var tlsConfig *tls.Config
	if settings.RedisTLSEnabled {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	client := redis.NewClient(&redis.Options{
		Addr:         settings.RedisAddress,
		Password:     settings.RedisPassword,
		DB:           settings.RedisDB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		TLSConfig:    tlsConfig,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis 失败: %w", err)
	}
	m.client = client
	m.signature = signature
	return client, nil
}

func normalizeDispatchLimit(rpmLimit int, safetyFactor float64, retryReserveRatio float64) int {
	if rpmLimit <= 0 {
		rpmLimit = 1
	}
	if safetyFactor <= 0 || safetyFactor > 1 {
		safetyFactor = 1
	}
	if retryReserveRatio < 0 || retryReserveRatio > 1 {
		retryReserveRatio = 0
	}
	effective := int(float64(rpmLimit) * safetyFactor * (1 - retryReserveRatio))
	if effective <= 0 {
		return 1
	}
	return effective
}

func normalizedReservationKey(value string) string {
	if strings.TrimSpace(value) == "" {
		return fmt.Sprintf("reservation-%d", time.Now().UnixNano())
	}
	return strings.TrimSpace(value)
}

func toInt64(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case string:
		parsed, _ := strconv.ParseInt(typed, 10, 64)
		return parsed
	default:
		return 0
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func max64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
