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
	Last   time.Time
}

func New(burst int, rate float64) *Limiter {
	if burst < 0 || rate < 0 {
		panic("burst/rate is negative")
	}
	return &Limiter{
		Burst:  float64(burst),
		Rate:   rate,
		Tokens: float64(burst),
		Last:   time.Now(),
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
		sinceLast := time.Since(l.Last).Seconds()
		l.Tokens = min(l.Tokens+sinceLast*l.Rate, float64(l.Burst))
		if l.Tokens >= n {
			l.Tokens -= n
			l.Last = time.Now()
			return true
		}
		return false
	}
}

func (l *Limiter) Allow() bool {
	return l.AllowN(1)
}

type Client struct {
	Limiter     *Limiter
	LastAccesed time.Time
}

type Shard struct {
	sync.Mutex
	ID      int
	Clients []Client
}

func NewShard(id int) *Shard {
	return &Shard{
		ID:      id,
		Clients: make([]Client, 0),
	}
}

func NewNShards(n int) []*Shard {
	shards := make([]*Shard, 0, n)
	for i := range n {
		s := NewShard(i)
		shards = append(shards, s)
	}
	return shards
}
