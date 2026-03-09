package ratelimiter

import (
	"context"
	"hash/fnv"
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
		//WARN: Panic used but i think this is fine
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
	Clients map[string]*Client
}

// Do not update ShardStore data
// after creation
type ShardStore []*Shard

func NewShard(id int) *Shard {
	return &Shard{
		ID:      id,
		Clients: make(map[string]*Client),
	}
}

// Takes the lock of the shard to and unlocks at end
func (s *Shard) CleanShard() {
	s.Lock()
	defer s.Unlock()
	for ip, cl := range s.Clients {
		if time.Since(cl.LastAccesed) > 3*time.Minute {
			delete(s.Clients, ip)
		}
	}
}

func NewNShards(n int) ShardStore {
	shards := make([]*Shard, 0, n)
	for i := range n {
		s := NewShard(i)
		shards = append(shards, s)
	}
	return shards
}

// WARN: Check if it updates orginal and muterx is not copies
// Prolly should not as mutex has noCopy struct
func (ss ShardStore) CleanShardStore(ctx context.Context) {
	maxTurn := len(ss)
	currTurn := 0
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			ss[currTurn].CleanShard()
			currTurn = (currTurn + 1) % maxTurn
		case <-ctx.Done():
			return
		}
	}
}

func (ss ShardStore) GetShardFromIP(ip string) (*Shard, error) {
	f := fnv.New64a()
	_, err := f.Write([]byte(ip))
	if err != nil {
		return nil, err
	}
	h := f.Sum64()
	return ss[int(h%uint64(len(ss)))], nil
}
