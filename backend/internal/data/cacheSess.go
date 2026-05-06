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

// NOTE: Not sure if mutex lock should be promted with embeddeding.
type CachedSess struct {
	users      map[string]*CachedUser
	ttl        time.Duration
	head       *CachedUser
	tail       *CachedUser
	entryNo    int
	maxEntries int
	cleanDur   time.Duration
	mut        sync.Mutex
}

func NewCache(ttl time.Duration, cleanDur time.Duration, maxEntries int) *CachedSess {
	head := &CachedUser{}
	tail := &CachedUser{}
	head.next = tail
	tail.prev = head

	return &CachedSess{
		users:      map[string]*CachedUser{},
		ttl:        ttl,
		head:       head,
		tail:       tail,
		cleanDur:   cleanDur,
		maxEntries: maxEntries,
	}
}

// putTop moves cu to the most-recently-used position.
// should have acquired a lock before using this unless cu is not yet shared.
func (cs *CachedSess) putTop(cu *CachedUser) {
	if cu == nil || cu == cs.head || cu == cs.tail {
		return
	}
	if cu.prev != nil {
		cu.prev.next = cu.next
	}
	if cu.next != nil {
		cu.next.prev = cu.prev
	}
	cu.prev = cs.head
	cu.next = cs.head.next
	cs.head.next.prev = cu
	cs.head.next = cu

	cu.lastUsed = time.Now()
}

func (cs *CachedSess) getUser(token string) (*User, bool) {
	cs.mut.Lock()
	defer cs.mut.Unlock()
	user, ok := cs.users[token]
	if !ok {
		return nil, false
	}
	cs.putTop(user)
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
	return cUser.User, cUser.Data, cUser.Scope, true
}

// remove a user from the lru cache double linked list
// should have acquired a lock before using this
func (cs *CachedSess) removeUser(cu *CachedUser) {
	if cu == nil || cu == cs.head || cu == cs.tail || cu.prev == nil || cu.next == nil {
		return
	}
	cu.prev.next = cu.next
	cu.next.prev = cu.prev
	cu.prev = nil
	cu.next = nil
	if _, ok := cs.users[cu.token]; ok {
		delete(cs.users, cu.token)
		cs.entryNo--
	}
}

// should have acquired a lock before using this
func (cs *CachedSess) removeLastUser() {
	last := cs.tail.prev
	cs.removeUser(last)
}

// call only if user not in cache
func (cs *CachedSess) setUser(token string, user *User, data map[string]any, scope Scope) error {
	if data == nil {
		return errors.New("dataMap is nil")
	}
	if cs.maxEntries <= 0 {
		return nil
	}
	cs.mut.Lock()
	defer cs.mut.Unlock()
	if cs.entryNo >= cs.maxEntries {
		cs.removeLastUser()
	}
	cu := &CachedUser{
		User:  user,
		token: token,
		Data:  data,
		Scope: scope,
	}
	cs.entryNo++
	cs.users[token] = cu
	cs.putTop(cu)
	return nil
}

func (cs *CachedSess) clean(ctx context.Context) {
	ticker := time.NewTicker(cs.cleanDur)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cs.mut.Lock()
			cUsers := cs.tail.prev
			for cUsers != cs.head {
				prev := cUsers.prev
				if time.Since(cUsers.lastUsed) > cs.ttl {
					cs.removeUser(cUsers)
				} else {
					break
				}
				cUsers = prev
			}
			cs.mut.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
