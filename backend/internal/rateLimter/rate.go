package ratelimiter

import (
	"math"
	"sync"
	"time"
)

const INF = math.MaxFloat64

// Token bucket algorithm
type Limiter struct {
	sync.Mutex
	Burst  float64
	Rate   float64
	Tokens float64
	last   time.Time
}

func New(burst int, rate float64) *Limiter {
	if burst < 0 || rate < 0 {
		panic("burst/rate is negative")
	}
	return &Limiter{
		Burst:  float64(burst),
		Rate:   rate,
		Tokens: float64(burst),
		last:   time.Now(),
	}
}

func (l *Limiter) AllowN(n float64) bool {
	if n < 0 {
		panic("tokens are negative")
	}
	l.Lock()
	defer l.Unlock()
	switch l.Burst {
	case INF:
		return true
	case 0:
		return false
	default:
		sinceLast := time.Since(l.last).Seconds()
		l.Tokens = min(l.Tokens+sinceLast*l.Rate, float64(l.Burst))
		if l.Tokens >= n {
			l.Tokens -= n
			l.last = time.Now()
			return true
		}
		return false
	}
}

func (l *Limiter) Allow() bool {
	return l.AllowN(1)
}
