package data

import (
	"context"
	"sync"
	"time"
)

type CachedUser struct {
	User     *User
	Data     map[string]any
	lastUsed time.Time
}

func NewCache(ttl time.Duration) *CachedSess {
	return &CachedSess{
		users: map[string]*CachedUser{},
		ttl:   5 * ttl,
	}
}

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
	user.lastUsed = time.Now()
	return user.User, true
}

// dataMap will not be nil
func (cs *CachedSess) getData(token string) (map[string]any, bool) {
	cs.RLock()
	defer cs.RUnlock()
	cUser, ok := cs.users[token]
	if !ok {
		return nil, false
	}
	cUser.lastUsed = time.Now()
	return cUser.Data, true
}

// dataMap will not be nil
func (cs *CachedSess) getUserAndData(token string) (*User, map[string]any, bool) {
	cs.RLock()
	defer cs.RUnlock()
	cUser, ok := cs.users[token]
	if !ok {
		return nil, nil, false
	}
	cUser.lastUsed = time.Now()
	return cUser.User, cUser.Data, true
}

func (cs *CachedSess) setUser(token string, user *User, data map[string]any) {
	if data == nil {
		//WARN: can panic here
		panic("data should be never nil should set empty val if nil")
	}
	cs.Lock()
	defer cs.Unlock()
	cs.users[token] = &CachedUser{
		User:     user,
		Data:     data,
		lastUsed: time.Now(),
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
