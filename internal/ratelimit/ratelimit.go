package ratelimit

import (
	"sync"

	"golang.org/x/time/rate"
)

const REQNUM = 0
const BURST = 4

type RateLimiter struct {
	sync.RWMutex
	Clients map[string]*rate.Limiter
}

func NewRateLimiter() *RateLimiter { return &RateLimiter{Clients: make(map[string]*rate.Limiter, 0)} }

func (rl *RateLimiter) GetClientRateLimit(ip string) bool {
	rl.RLock()
	clientRl, exists := rl.Clients[ip]
	rl.RUnlock()
	if exists {
		return clientRl.Allow()

	}

	rl.Lock()
	rl.Clients[ip] = rate.NewLimiter(REQNUM, BURST)
	rl.Unlock()
	return true
}

func (rl *RateLimiter) GetTokenAmount(ip string) float64 {
	rl.RLock()
	clientRl, exists := rl.Clients[ip]
	rl.RUnlock()
	if exists {
		return clientRl.Tokens()
	}
	return 0.0
}
