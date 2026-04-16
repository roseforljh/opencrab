package domain

import "context"

type DispatchRuntimeSettings struct {
	RedisEnabled      bool
	RedisAddress      string
	RedisPassword     string
	RedisDB           int
	RedisTLSEnabled   bool
	RedisKeyPrefix    string
	RetryReserveRatio float64
	WorkerConcurrency int
	PauseDispatch     bool
	MaxAttempts       int
	BackoffDelayMs    int
	SyncHoldMs        int
}

type DispatchRuntimeConfigStore interface {
	GetDispatchRuntimeSettings(ctx context.Context) (DispatchRuntimeSettings, error)
}

type DispatchReservationInput struct {
	ChannelName     string  `json:"channel_name"`
	RPMLimit        int     `json:"rpm_limit"`
	MaxInflight     int     `json:"max_inflight"`
	SafetyFactor    float64 `json:"safety_factor"`
	LeaseMs         int64   `json:"lease_ms"`
	CooldownUntilMs int64   `json:"cooldown_until_ms"`
	HealthPenaltyMs int64   `json:"health_penalty_ms"`
	StickyBiasMs    int64   `json:"sticky_bias_ms"`
	ReservationKey  string  `json:"reservation_key"`
}

type DispatchReservationResult struct {
	ChannelName       string `json:"channel_name"`
	ReservationKey    string `json:"reservation_key"`
	DispatchAtMs      int64  `json:"dispatch_at_ms"`
	WaitMs            int64  `json:"wait_ms"`
	CurrentInflight   int64  `json:"current_inflight"`
	ReservedTatMs     int64  `json:"reserved_tat_ms"`
	InflightReadyAtMs int64  `json:"inflight_ready_at_ms"`
	LeaseAcquired     bool   `json:"lease_acquired"`
	Runtime           string `json:"runtime"`
}

type DispatchReleaseInput struct {
	ChannelName    string
	ReservationKey string
}

type DispatchQuotaManager interface {
	Reserve(ctx context.Context, input DispatchReservationInput) (DispatchReservationResult, error)
	Release(ctx context.Context, input DispatchReleaseInput) error
}
