package data

import (
	"context"
	"errors"
	"sync"
	"time"
)

// WARN: Prolly need to use atomic.Int instead of lock for [CachedUser.lastUsed]
// need to profile then to
// TODO: Make the cache generic instead (do it when more cache is needed ig)
// TODO: May be make cache into interface to facilitate multiple cache implementations like redis
type CachedUser struct {
	token    string //key of map needed to remove the last user from map
	User     *User
	Data     map[string]any
	next     *CachedUser
	prev     *CachedUser
	lastUsed time.Time
	Scope    Scope
}

func makeNullCacheUser() *CachedUser {
	cu := &CachedUser{}
	cu.next = cu
	cu.prev = cu
	return cu
}

// should this be in a init instead not sure??
var NullUser = makeNullCacheUser()

// NOTE: Not sure if mutex lock should be promted with embeddeding.
type CachedSess struct {
	users      map[string]*CachedUser
	ttl        time.Duration
	head       *CachedUser
	tail       *CachedUser
	entryNo    int
	maxEntries int
	mut        sync.Mutex
}

func NewCache(ttl time.Duration, maxEntries int) *CachedSess {
	return &CachedSess{
		users:      map[string]*CachedUser{},
		ttl:        ttl,
		head:       &CachedUser{},
		tail:       &CachedUser{},
		maxEntries: maxEntries,
	}
}

// TODO: test this
func (cs *CachedSess) putTop(cu *CachedUser) {
	if cu == nil {
		return
	}
	cu.prev.next = cu.next
	cu.next = cs.head.next
	cs.head.next = cu
}

func (cs *CachedSess) getUser(token string) (*User, bool) {
	cs.mut.Lock()
	defer cs.mut.Unlock()
	user, ok := cs.users[token]
	if !ok {
		return nil, false
	}
	cs.putTop(user)
	user.lastUsed = time.Now()
	return user.User, true
}

// dataMap will not be nil if present in cache
func (cs *CachedSess) getData(token string) (map[string]any, bool) {
	cs.mut.Lock()
	defer cs.mut.Unlock()
	cUser, ok := cs.users[token]
	if !ok {
		return nil, false
	}
	cs.putTop(cUser)
	cUser.lastUsed = time.Now()
	return cUser.Data, true
}

// dataMap will not be nil if present in cache.
// return the user,their session data and token scope
func (cs *CachedSess) getUserAndData(token string) (*User, map[string]any, Scope, bool) {
	cs.mut.Lock()
	defer cs.mut.Unlock()
	cUser, ok := cs.users[token]
	if !ok {
		return nil, nil, ScopeUnknown, false
	}
	cs.putTop(cUser)
	cUser.lastUsed = time.Now()
	return cUser.User, cUser.Data, cUser.Scope, true
}

// remove a user from the lru cache double linked list
// should not cause nullptr deref due to null struct
// should have acquired a lock before using this
func (cs *CachedSess) removeUser(cu *CachedUser) {
	if cu != cs.head || cu != cs.tail {
		cu.prev.next = cu.next
		delete(cs.users, cu.token)
	}
}

// should have acquired a lock before using this
func (cs *CachedSess) removeLastUser() {
	last := cs.tail.prev
	cs.removeUser(last)
}

// only called if not found in cache
func (cs *CachedSess) setUser(token string, user *User, data map[string]any, scope Scope) error {
	if data == nil {
		return errors.New("dataMap is nil")
	}
	cs.mut.Lock()
	defer cs.mut.Unlock()
	if cs.entryNo+1 > cs.maxEntries {
		cs.removeLastUser()
	}
	cu := &CachedUser{
		User:     user,
		token:    token,
		Data:     data,
		lastUsed: time.Now(),
		Scope:    scope,
		next:     NullUser,
		prev:     NullUser,
	}
	cs.entryNo++
	cs.users[token] = cu
	cs.putTop(cu)
	return nil
}

func (cs *CachedSess) clean(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ticker.C:
			cs.mut.Lock()
			for token, cUsers := range cs.users {
				if time.Since(cUsers.lastUsed) > cs.ttl {
					delete(cs.users, token)
					cs.removeUser(cUsers)
				}
			}
			cs.mut.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
