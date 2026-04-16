package usecase

import (
	"context"
	"testing"

	"opencrab/internal/domain"

	miniredis "github.com/alicebob/miniredis/v2"
)

type fakeDispatchRuntimeConfigStore struct {
	settings domain.DispatchRuntimeSettings
	err      error
}

func (s fakeDispatchRuntimeConfigStore) GetDispatchRuntimeSettings(ctx context.Context) (domain.DispatchRuntimeSettings, error) {
	return s.settings, s.err
}

func TestRedisDispatchQuotaManagerDisabledFallsBack(t *testing.T) {
	manager := NewRedisDispatchQuotaManager(fakeDispatchRuntimeConfigStore{settings: domain.DispatchRuntimeSettings{RedisEnabled: false}})
	result, err := manager.Reserve(context.Background(), domain.DispatchReservationInput{ChannelName: "c1", RPMLimit: 1000, MaxInflight: 16, SafetyFactor: 0.9, LeaseMs: 1000})
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if !result.LeaseAcquired || result.Runtime != "disabled" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestRedisDispatchQuotaManagerReservesSequentialRPM(t *testing.T) {
	server := miniredis.RunT(t)
	manager := NewRedisDispatchQuotaManager(fakeDispatchRuntimeConfigStore{settings: domain.DispatchRuntimeSettings{RedisEnabled: true, RedisAddress: server.Addr(), RedisDB: 0, RedisKeyPrefix: "test", RetryReserveRatio: 0}})
	first, err := manager.Reserve(context.Background(), domain.DispatchReservationInput{ChannelName: "c1", RPMLimit: 60, MaxInflight: 4, SafetyFactor: 1, LeaseMs: 1000, ReservationKey: "r1"})
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	second, err := manager.Reserve(context.Background(), domain.DispatchReservationInput{ChannelName: "c1", RPMLimit: 60, MaxInflight: 4, SafetyFactor: 1, LeaseMs: 1000, ReservationKey: "r2"})
	if err != nil {
		t.Fatalf("second reserve: %v", err)
	}
	if !first.LeaseAcquired || second.DispatchAtMs <= first.DispatchAtMs || second.WaitMs <= 0 {
		t.Fatalf("unexpected sequential reservation: first=%#v second=%#v", first, second)
	}
}

func TestRedisDispatchQuotaManagerRespectsInflightLimit(t *testing.T) {
	server := miniredis.RunT(t)
	manager := NewRedisDispatchQuotaManager(fakeDispatchRuntimeConfigStore{settings: domain.DispatchRuntimeSettings{RedisEnabled: true, RedisAddress: server.Addr(), RedisDB: 0, RedisKeyPrefix: "test", RetryReserveRatio: 0}})
	first, err := manager.Reserve(context.Background(), domain.DispatchReservationInput{ChannelName: "c1", RPMLimit: 60000, MaxInflight: 1, SafetyFactor: 1, LeaseMs: 5000, ReservationKey: "lease-1"})
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	second, err := manager.Reserve(context.Background(), domain.DispatchReservationInput{ChannelName: "c1", RPMLimit: 60000, MaxInflight: 1, SafetyFactor: 1, LeaseMs: 5000, ReservationKey: "lease-2"})
	if err != nil {
		t.Fatalf("second reserve: %v", err)
	}
	if !first.LeaseAcquired || second.LeaseAcquired || second.InflightReadyAtMs <= first.DispatchAtMs {
		t.Fatalf("unexpected inflight reservation: first=%#v second=%#v", first, second)
	}
}

func TestRedisDispatchQuotaManagerReleaseRemovesLease(t *testing.T) {
	server := miniredis.RunT(t)
	manager := NewRedisDispatchQuotaManager(fakeDispatchRuntimeConfigStore{settings: domain.DispatchRuntimeSettings{RedisEnabled: true, RedisAddress: server.Addr(), RedisDB: 0, RedisKeyPrefix: "test", RetryReserveRatio: 0}})
	first, err := manager.Reserve(context.Background(), domain.DispatchReservationInput{ChannelName: "c1", RPMLimit: 60000, MaxInflight: 1, SafetyFactor: 1, LeaseMs: 5000, ReservationKey: "lease-1"})
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	if err := manager.Release(context.Background(), domain.DispatchReleaseInput{ChannelName: "c1", ReservationKey: first.ReservationKey}); err != nil {
		t.Fatalf("release: %v", err)
	}
	second, err := manager.Reserve(context.Background(), domain.DispatchReservationInput{ChannelName: "c1", RPMLimit: 60000, MaxInflight: 1, SafetyFactor: 1, LeaseMs: 5000, ReservationKey: "lease-2"})
	if err != nil {
		t.Fatalf("second reserve: %v", err)
	}
	if !second.LeaseAcquired {
		t.Fatalf("expected lease after release: %#v", second)
	}
}

func TestRedisDispatchQuotaManagerDegradesWhenRedisUnavailable(t *testing.T) {
	manager := NewRedisDispatchQuotaManager(fakeDispatchRuntimeConfigStore{settings: domain.DispatchRuntimeSettings{RedisEnabled: true, RedisAddress: "127.0.0.1:1", RedisDB: 0, RedisKeyPrefix: "test", RetryReserveRatio: 0}})
	result, err := manager.Reserve(context.Background(), domain.DispatchReservationInput{ChannelName: "c1", RPMLimit: 1000, MaxInflight: 16, SafetyFactor: 1, LeaseMs: 1000, ReservationKey: "lease-1"})
	if err != nil {
		t.Fatalf("reserve should degrade, got err: %v", err)
	}
	if !result.LeaseAcquired || result.Runtime != "redis_unavailable" {
		t.Fatalf("unexpected degraded result: %#v", result)
	}
}
