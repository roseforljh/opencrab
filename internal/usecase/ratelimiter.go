package usecase

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*limiterEntry
	rate    rate.Limit
	burst   int
}

func NewRateLimiter(perSecond int, burst int) *RateLimiter {
	if perSecond <= 0 {
		perSecond = 5
	}
	if burst <= 0 {
		burst = 10
	}

	return &RateLimiter{
		entries: make(map[string]*limiterEntry),
		rate:    rate.Limit(perSecond),
		burst:   burst,
	}
}

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entry, ok := r.entries[key]
	if !ok {
		entry = &limiterEntry{limiter: rate.NewLimiter(r.rate, r.burst), lastAccess: now}
		r.entries[key] = entry
	}

	entry.lastAccess = now
	r.cleanup(now)

	return entry.limiter.Allow()
}

func (r *RateLimiter) cleanup(now time.Time) {
	for key, entry := range r.entries {
		if now.Sub(entry.lastAccess) > 15*time.Minute {
			delete(r.entries, key)
		}
	}
}
