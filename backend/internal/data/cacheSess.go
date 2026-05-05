package data

import (
	"context"
	"sync"
	"time"
)

// WARN: Prolly need to use atomic.Int instead of lock for [CachedUser.lastUsed]
// need to profile then to
type CachedUser struct {
	User     *User
	Data     map[string]any
	lastUsed time.Time
	Scope    Scope
}

func NewCache(ttl time.Duration) *CachedSess {
	return &CachedSess{
		users: map[string]*CachedUser{},
		ttl:   ttl,
	}
}

// NOTE: Not sure if mutex lock should be promted with embeddeding.
type CachedSess struct {
	users map[string]*CachedUser
	ttl   time.Duration
	sync.RWMutex
}

func (cs *CachedSess) getUser(token string) (*User, bool) {
	cs.RLock()
	defer cs.RUnlock()
	user, ok := cs.users[token]
	if !ok {
		return nil, false
	}
	//WARN: Race cond but dont think it matters cause like if a user had accessed session with
	// time period close enough for a race cond, it wouldnt matter which won
	user.lastUsed = time.Now()
	return user.User, true
}

// dataMap will not be nil if present in cache
func (cs *CachedSess) getData(token string) (map[string]any, bool) {
	cs.RLock()
	defer cs.RUnlock()
	cUser, ok := cs.users[token]
	if !ok {
		return nil, false
	}
	//WARN: Race cond with same reason above on why its prolly is fine
	cUser.lastUsed = time.Now()
	return cUser.Data, true
}

// dataMap will not be nil if present in cache.
// return the user,their session data and token scope
func (cs *CachedSess) getUserAndData(token string) (*User, map[string]any, Scope, bool) {
	cs.RLock()
	defer cs.RUnlock()
	cUser, ok := cs.users[token]
	if !ok {
		return nil, nil, ScopeUnknown, false
	}
	//WARN: Race cond with same reason above on why its prolly is fine
	cUser.lastUsed = time.Now()
	return cUser.User, cUser.Data, cUser.Scope, true
}

func (cs *CachedSess) setUser(token string, user *User, data map[string]any, scope Scope) {
	if data == nil {
		//WARN: Race cond with same reason above on why its prolly is fine
		panic("data should be never nil should set empty val if nil")
	}
	cs.Lock()
	defer cs.Unlock()
	cs.users[token] = &CachedUser{
		User:     user,
		Data:     data,
		lastUsed: time.Now(),
		Scope:    scope,
	}
}

func (cs *CachedSess) clean(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ticker.C:
			cs.Lock()
			for token, cUsers := range cs.users {
				if time.Since(cUsers.lastUsed) > cs.ttl {
					delete(cs.users, token)
				}
			}
			cs.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
