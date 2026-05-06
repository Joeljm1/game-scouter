package data

import (
	"testing"
	"time"
)

func TestCachedSess_putTop(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		ttl        time.Duration
		maxEntries int
		// Named input parameters for target function.
		cu *CachedUser
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewCache(tt.ttl, tt.maxEntries)
			cs.putTop(tt.cu)
		})
	}
}

// TODO: test datamap nil feature
func TestCachedSess_setUser(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		ttl        time.Duration
		maxEntries int
		// Named input parameters for target function.
		token   string
		user    *User
		data    map[string]any
		scope   Scope
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewCache(tt.ttl, tt.maxEntries)
			gotErr := cs.setUser(tt.token, tt.user, tt.data, tt.scope)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("setUser() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("setUser() succeeded unexpectedly")
			}
		})
	}
}
