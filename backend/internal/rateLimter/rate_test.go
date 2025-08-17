package ratelimiter

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		burst int
		rate  float64
	}{
		{"basic limiter", 10, 5.0},
		{"zero burst", 0, 1.0},
		{"high rate", 100, 1000.0},
		{"fractional rate", 5, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := New(tt.burst, tt.rate)

			if limiter.Burst != float64(tt.burst) {
				t.Errorf("New().Burst = %v, want %v", limiter.Burst, float64(tt.burst))
			}
			if limiter.Rate != tt.rate {
				t.Errorf("New().Rate = %v, want %v", limiter.Rate, tt.rate)
			}
			if limiter.Tokens != float64(tt.burst) {
				t.Errorf("New().Tokens = %v, want %v", limiter.Tokens, float64(tt.burst))
			}
			if time.Since(limiter.last) > time.Millisecond {
				t.Errorf("New().last should be recent, but was %v ago", time.Since(limiter.last))
			}
		})
	}
}

func TestAllow(t *testing.T) {
	t.Run("basic allow", func(t *testing.T) {
		limiter := New(5, 1.0)

		// Should allow up to burst capacity
		for i := 0; i < 5; i++ {
			if !limiter.Allow() {
				t.Errorf("Allow() should return true for request %d", i+1)
			}
		}

		// Should deny next request (bucket empty)
		if limiter.Allow() {
			t.Errorf("Allow() should return false when bucket is empty")
		}
	})
}

func TestAllowN(t *testing.T) {
	t.Run("basic allowN", func(t *testing.T) {
		limiter := New(10, 1.0)

		// Allow 3 tokens
		if !limiter.AllowN(3) {
			t.Errorf("AllowN(3) should return true")
		}

		// Allow 7 more tokens (should succeed, exactly empties bucket)
		if !limiter.AllowN(7) {
			t.Errorf("AllowN(7) should return true")
		}

		// Should deny 1 more token (bucket empty)
		if limiter.AllowN(1) {
			t.Errorf("AllowN(1) should return false when bucket is empty")
		}
	})

	t.Run("request more than available", func(t *testing.T) {
		limiter := New(5, 1.0)

		// Request more tokens than available
		if limiter.AllowN(10) {
			t.Errorf("AllowN(10) should return false when only 5 tokens available")
		}

		// Bucket should still have original tokens
		if !limiter.AllowN(5) {
			t.Errorf("AllowN(5) should still return true after failed large request")
		}
	})
}

func TestBurstLimit(t *testing.T) {
	t.Run("enforce burst limit", func(t *testing.T) {
		limiter := New(3, 1.0)

		// Consume all tokens
		for i := 0; i < 3; i++ {
			if !limiter.Allow() {
				t.Errorf("Allow() should return true for request %d", i+1)
			}
		}

		// Wait for tokens to refill beyond burst capacity
		time.Sleep(5 * time.Second)

		// Should only have burst capacity, not more
		if !limiter.AllowN(3) {
			t.Errorf("AllowN(3) should return true after refill")
		}

		if limiter.AllowN(1) {
			t.Errorf("AllowN(1) should return false, tokens should be capped at burst")
		}
	})
}

func TestRateLimiting(t *testing.T) {
	t.Run("rate limiting over time", func(t *testing.T) {
		// 2 tokens/second rate
		limiter := New(1, 2.0)

		// Consume initial token
		if !limiter.Allow() {
			t.Errorf("Initial Allow() should return true")
		}

		// Should be denied immediately
		if limiter.Allow() {
			t.Errorf("Second Allow() should return false immediately")
		}

		// Wait half a second (should get 1 token)
		time.Sleep(500 * time.Millisecond)

		if !limiter.Allow() {
			t.Errorf("Allow() should return true after 0.5s with 2 tokens/sec rate")
		}

		// Should be denied again
		if limiter.Allow() {
			t.Errorf("Allow() should return false immediately after consuming refilled token")
		}
	})
}

func TestInfiniteBurst(t *testing.T) {
	t.Run("infinite burst case", func(t *testing.T) {
		limiter := &Limiter{
			Burst:  INF,
			Rate:   1.0,
			Tokens: INF,
			last:   time.Now(),
		}

		// Should always allow regardless of tokens requested
		for i := 0; i < 1000; i++ {
			if !limiter.Allow() {
				t.Errorf("Allow() should always return true with infinite burst")
			}
		}

		if !limiter.AllowN(1000000) {
			t.Errorf("AllowN(1000000) should return true with infinite burst")
		}
	})
}

func TestZeroBurst(t *testing.T) {
	t.Run("zero burst case", func(t *testing.T) {
		limiter := &Limiter{
			Burst:  0,
			Rate:   1.0,
			Tokens: 0,
			last:   time.Now(),
		}

		// Should never allow any tokens
		if limiter.Allow() {
			t.Errorf("Allow() should return false with zero burst")
		}

		if limiter.AllowN(1) {
			t.Errorf("AllowN(1) should return false with zero burst")
		}

		// Even after waiting
		time.Sleep(100 * time.Millisecond)
		if limiter.Allow() {
			t.Errorf("Allow() should return false with zero burst even after waiting")
		}
	})
}

func TestTokenRefill(t *testing.T) {
	t.Run("token refill mechanism", func(t *testing.T) {
		// 10 tokens/second rate, burst of 5
		limiter := New(5, 10.0)

		// Consume all tokens
		for i := 0; i < 5; i++ {
			if !limiter.Allow() {
				t.Errorf("Allow() should return true for request %d", i+1)
			}
		}

		// Wait 0.3 seconds (should get 3 tokens: 10 * 0.3 = 3)
		time.Sleep(300 * time.Millisecond)

		// Should be able to get 3 tokens
		if !limiter.AllowN(3) {
			t.Errorf("AllowN(3) should return true after 0.3s refill")
		}

		// Should not be able to get 1 more
		if limiter.AllowN(1) {
			t.Errorf("AllowN(1) should return false after consuming refilled tokens")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	t.Run("concurrent access safety", func(t *testing.T) {
		limiter := New(1000, 100.0)

		var wg sync.WaitGroup
		successCount := int64(0)
		numGoroutines := 50
		requestsPerGoroutine := 100

		var mu sync.Mutex

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				localSuccess := 0
				for j := 0; j < requestsPerGoroutine; j++ {
					if limiter.Allow() {
						localSuccess++
					}
				}
				mu.Lock()
				successCount += int64(localSuccess)
				mu.Unlock()
			}()
		}

		wg.Wait()

		// Should have some successes but not more than burst capacity + some refill
		if successCount == 0 {
			t.Errorf("Expected some successful requests, got 0")
		}

		// Allow some tolerance for token refill during concurrent execution
		// The test takes some time to run, so some tokens may refill
		if successCount > 1100 {
			t.Errorf("Expected at most ~1100 successes (burst + some refill), got %d", successCount)
		}

		t.Logf("Concurrent test: %d successful requests out of %d total",
			successCount, numGoroutines*requestsPerGoroutine)
	})
}

func TestFractionalTokens(t *testing.T) {
	t.Run("fractional token requests", func(t *testing.T) {
		limiter := New(5, 1.0)

		// Request fractional tokens
		if !limiter.AllowN(2.5) {
			t.Errorf("AllowN(2.5) should return true")
		}

		if !limiter.AllowN(2.5) {
			t.Errorf("AllowN(2.5) should return true for second request")
		}

		// Should not allow more (2.5 + 2.5 = 5, bucket empty)
		if limiter.AllowN(0.1) {
			t.Errorf("AllowN(0.1) should return false when bucket is empty")
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("zero token request", func(t *testing.T) {
		limiter := New(1, 1.0)

		// Requesting 0 tokens should always succeed
		if !limiter.AllowN(0) {
			t.Errorf("AllowN(0) should always return true")
		}

		// Should not consume any tokens
		if !limiter.Allow() {
			t.Errorf("Allow() should still return true after AllowN(0)")
		}
	})

	t.Run("very high rate", func(t *testing.T) {
		limiter := New(1, 1000000.0) // 1M tokens per second

		// Consume initial token
		if !limiter.Allow() {
			t.Errorf("Initial Allow() should return true")
		}

		// Should refill very quickly
		time.Sleep(10 * time.Millisecond) // 0.01 seconds * 1M = 10K tokens

		if !limiter.Allow() {
			t.Errorf("Allow() should return true with very high refill rate")
		}
	})
}

// Benchmark tests
func BenchmarkAllow(b *testing.B) {
	limiter := New(1000000, 1000000.0) // Large capacity to avoid blocking

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow()
		}
	})
}

func BenchmarkAllowN(b *testing.B) {
	limiter := New(1000000, 1000000.0) // Large capacity to avoid blocking

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.AllowN(1)
		}
	})
}
