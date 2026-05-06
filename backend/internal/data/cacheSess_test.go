package data

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestCachedSess_putTop(t *testing.T) {
	cs := NewCache(time.Minute, 2*time.Minute, 3)
	users := []*CachedUser{
		{token: "first"},
		{token: "second"},
		{token: "third"},
	}
	for _, user := range users {
		cs.users[user.token] = user
		cs.entryNo++
		cs.putTop(user)
	}

	assertCacheOrder(t, cs, []string{"third", "second", "first"})

	cs.putTop(users[0])

	assertCacheOrder(t, cs, []string{"first", "third", "second"})
	assertCacheLinks(t, cs)

	cs.putTop(nil)
	cs.putTop(cs.head)
	cs.putTop(cs.tail)

	assertCacheOrder(t, cs, []string{"first", "third", "second"})

	cs.putTop(cs.head.next.next)
	assertCacheOrder(t, cs, []string{"third", "first", "second"})
	assertCacheLinks(t, cs)
}

func TestCachedSess_setUserRequiresNonNilData(t *testing.T) {
	cs := NewCache(time.Minute, 2*time.Minute, 1)

	if err := cs.setUser("token", &User{ID: 1}, nil, ScopeAuthentication); err == nil {
		t.Fatal("setUser() succeeded with nil data")
	}
	if len(cs.users) != 0 || cs.entryNo != 0 {
		t.Fatalf("setUser() cached nil data entry: len=%d entryNo=%d", len(cs.users), cs.entryNo)
	}
}

func TestCachedSess_setUserAndGetUserAndData(t *testing.T) {
	cs := NewCache(time.Minute, 2*time.Minute, 2)
	user := &User{ID: 1, Name: "joel"}
	data := map[string]any{"step": "signup"}

	if err := cs.setUser("token", user, data, ScopeAuthentication); err != nil {
		t.Fatalf("setUser() failed: %v", err)
	}

	gotUser, gotData, gotScope, ok := cs.getUserAndData("token")
	if !ok {
		t.Fatal("getUserAndData() missed cached user")
	}
	if gotUser != user {
		t.Fatalf("getUserAndData() user = %p, want %p", gotUser, user)
	}
	if !reflect.DeepEqual(gotData, data) {
		t.Fatalf("getUserAndData() data = %#v, want %#v", gotData, data)
	}
	if gotScope != ScopeAuthentication {
		t.Fatalf("getUserAndData() scope = %q, want %q", gotScope, ScopeAuthentication)
	}
	assertCacheOrder(t, cs, []string{"token"})
	assertCacheLinks(t, cs)
}

func TestCachedSess_getUserAndGetDataMoveEntryToTop(t *testing.T) {
	cs := NewCache(time.Minute, 2*time.Minute, 3)
	mustSetCachedUser(t, cs, "one", 1, ScopeAuthentication, map[string]any{"n": 1})
	mustSetCachedUser(t, cs, "two", 2, ScopeOIDC, map[string]any{"n": 2})
	mustSetCachedUser(t, cs, "three", 3, ScopeActivation, map[string]any{"n": 3})

	gotUser, ok := cs.getUser("one")
	if !ok {
		t.Fatal("getUser() missed cached user")
	}
	if gotUser.ID != 1 {
		t.Fatalf("getUser() ID = %d, want 1", gotUser.ID)
	}
	assertCacheOrder(t, cs, []string{"one", "three", "two"})

	gotData, ok := cs.getData("two")
	if !ok {
		t.Fatal("getData() missed cached data")
	}
	if gotData["n"] != 2 {
		t.Fatalf("getData() n = %v, want 2", gotData["n"])
	}
	assertCacheOrder(t, cs, []string{"two", "one", "three"})
	assertCacheLinks(t, cs)
}

func TestCachedSess_getMisses(t *testing.T) {
	cs := NewCache(time.Minute, 2*time.Minute, 1)

	if gotUser, ok := cs.getUser("missing"); ok || gotUser != nil {
		t.Fatalf("getUser() = (%#v, %v), want (nil, false)", gotUser, ok)
	}
	if gotData, ok := cs.getData("missing"); ok || gotData != nil {
		t.Fatalf("getData() = (%#v, %v), want (nil, false)", gotData, ok)
	}
	if gotUser, gotData, gotScope, ok := cs.getUserAndData("missing"); ok || gotUser != nil || gotData != nil || gotScope != ScopeUnknown {
		t.Fatalf("getUserAndData() = (%#v, %#v, %q, %v), want (nil, nil, %q, false)", gotUser, gotData, gotScope, ok, ScopeUnknown)
	}
}

func TestCachedSess_EvictsLeastRecentlyUsedEntry(t *testing.T) {
	cs := NewCache(time.Minute, 2*time.Minute, 2)
	mustSetCachedUser(t, cs, "one", 1, ScopeAuthentication, map[string]any{"n": 1})
	mustSetCachedUser(t, cs, "two", 2, ScopeOIDC, map[string]any{"n": 2})

	if _, ok := cs.getUser("one"); !ok {
		t.Fatal("getUser() missed cached user")
	}
	mustSetCachedUser(t, cs, "three", 3, ScopeActivation, map[string]any{"n": 3})

	assertCacheOrder(t, cs, []string{"three", "one"})
	assertCacheLinks(t, cs)
	if _, ok := cs.getUser("two"); ok {
		t.Fatal("least recently used entry was not evicted")
	}
	if len(cs.users) != 2 || cs.entryNo != 2 {
		t.Fatalf("cache size = len %d entryNo %d, want 2/2", len(cs.users), cs.entryNo)
	}
}

func TestCachedSess_setUserUpdatesExistingEntry(t *testing.T) {
	cs := NewCache(time.Minute, 2*time.Minute, 2)
	mustSetCachedUser(t, cs, "one", 1, ScopeAuthentication, map[string]any{"n": 1})
	mustSetCachedUser(t, cs, "two", 2, ScopeOIDC, map[string]any{"n": 2})

	updatedData := map[string]any{"n": 10}
	if err := cs.setUser("one", &User{ID: 10}, updatedData, ScopeActivation); err != nil {
		t.Fatalf("setUser() failed: %v", err)
	}

	gotUser, gotData, gotScope, ok := cs.getUserAndData("one")
	if !ok {
		t.Fatal("getUserAndData() missed updated user")
	}
	if gotUser.ID != 10 {
		t.Fatalf("updated user ID = %d, want 10", gotUser.ID)
	}
	if !reflect.DeepEqual(gotData, updatedData) {
		t.Fatalf("updated data = %#v, want %#v", gotData, updatedData)
	}
	if gotScope != ScopeActivation {
		t.Fatalf("updated scope = %q, want %q", gotScope, ScopeActivation)
	}
	if len(cs.users) != 2 || cs.entryNo != 2 {
		t.Fatalf("cache size = len %d entryNo %d, want 2/2", len(cs.users), cs.entryNo)
	}
	assertCacheOrder(t, cs, []string{"one", "two"})
	assertCacheLinks(t, cs)
}

func TestCachedSess_setUserWithZeroCapacityDoesNotCache(t *testing.T) {
	cs := NewCache(time.Minute, 2*time.Minute, 0)
	if err := cs.setUser("token", &User{ID: 1}, map[string]any{}, ScopeAuthentication); err != nil {
		t.Fatalf("setUser() failed: %v", err)
	}
	if len(cs.users) != 0 || cs.entryNo != 0 {
		t.Fatalf("zero capacity cache size = len %d entryNo %d, want 0/0", len(cs.users), cs.entryNo)
	}
	assertCacheOrder(t, cs, []string{})
	assertCacheLinks(t, cs)
}

func TestCachedSess_removeLastUser(t *testing.T) {
	cs := NewCache(time.Minute, 2*time.Minute, 2)
	mustSetCachedUser(t, cs, "one", 1, ScopeAuthentication, map[string]any{"n": 1})
	mustSetCachedUser(t, cs, "two", 2, ScopeOIDC, map[string]any{"n": 2})

	cs.removeLastUser()

	assertCacheOrder(t, cs, []string{"two"})
	assertCacheLinks(t, cs)
	if _, ok := cs.users["one"]; ok {
		t.Fatal("removeLastUser() did not remove oldest entry")
	}
}

func TestCachedSess_cleanRemovesExpiredEntries(t *testing.T) {
	cs := NewCache(25*time.Millisecond, 5*time.Millisecond, 3)
	mustSetCachedUser(t, cs, "expired-one", 1, ScopeAuthentication, map[string]any{"n": 1})
	mustSetCachedUser(t, cs, "expired-two", 2, ScopeOIDC, map[string]any{"n": 2})
	mustSetCachedUser(t, cs, "fresh", 3, ScopeActivation, map[string]any{"n": 3})

	cs.mut.Lock()
	cs.users["expired-one"].lastUsed = time.Now().Add(-time.Minute)
	cs.users["expired-two"].lastUsed = time.Now().Add(-time.Minute)
	cs.users["fresh"].lastUsed = time.Now()
	cs.mut.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		cs.clean(ctx)
	}()

	deadline := time.After(200 * time.Millisecond)
	for {
		cs.mut.Lock()
		_, expiredOneOK := cs.users["expired-one"]
		_, expiredTwoOK := cs.users["expired-two"]
		_, freshOK := cs.users["fresh"]
		entryNo := cs.entryNo
		cs.mut.Unlock()

		if !expiredOneOK && !expiredTwoOK && freshOK && entryNo == 1 {
			break
		}

		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatalf("clean() did not remove only expired entries before timeout: expired-one=%v expired-two=%v fresh=%v entryNo=%d", expiredOneOK, expiredTwoOK, freshOK, entryNo)
		case <-time.After(time.Millisecond):
		}
	}

	cancel()
	<-done

	assertCacheOrder(t, cs, []string{"fresh"})
	assertCacheLinks(t, cs)
}

func mustSetCachedUser(t *testing.T, cs *CachedSess, token string, userID int64, scope Scope, data map[string]any) {
	t.Helper()
	if err := cs.setUser(token, &User{ID: userID}, data, scope); err != nil {
		t.Fatalf("setUser(%q) failed: %v", token, err)
	}
}

func assertCacheOrder(t *testing.T, cs *CachedSess, want []string) {
	t.Helper()

	var got = []string{}
	for user := cs.head.next; user != nil && user != cs.tail; user = user.next {
		got = append(got, user.token)
	}

	if len(got) != len(cs.users) {
		t.Fatalf("cache list has %d entries, users map has %d", len(got), len(cs.users))
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cache order = %#v, want %#v", got, want)
	}
}

func assertCacheLinks(t *testing.T, cs *CachedSess) {
	t.Helper()

	if cs.head.prev != nil {
		t.Fatal("head.prev should be nil")
	}
	if cs.tail.next != nil {
		t.Fatal("tail.next should be nil")
	}
	if cs.head.next == nil {
		t.Fatal("head.next should not be nil")
	}
	if cs.tail.prev == nil {
		t.Fatal("tail.prev should not be nil")
	}

	seen := 0
	for user := cs.head.next; user != cs.tail; user = user.next {
		if user == nil {
			t.Fatal("cache list ended before tail")
		}
		if user.next == nil {
			t.Fatalf("entry %q has nil next", user.token)
		}
		if user.next.prev != user {
			t.Fatalf("entry %q next.prev does not point back to it", user.token)
		}
		seen++
	}
	if seen != len(cs.users) {
		t.Fatalf("cache list has %d entries, users map has %d", seen, len(cs.users))
	}
	if cs.entryNo != seen {
		t.Fatalf("entryNo = %d, users map has %d", cs.entryNo, len(cs.users))
	}
}
