package main

import "sync"

var (
	userCache        map[int]User
	userAccountCache map[string]User
	userCachem       [100]sync.RWMutex
)

func init() {
	userCache = make(map[int]User)
	userAccountCache = make(map[string]User)

	for i := 0; i < 100; i++ {
		userCachem[i] = sync.RWMutex{}
	}
}

func setUser(user User) {
	userCachem[user.ID%100].Lock()
	defer userCachem[user.ID%100].Unlock()
	userAccountCache[user.AccountName] = user
	userCache[user.ID] = user
}

func fromAccount(name string) (user User, ok bool) {
	userCachem[user.ID%100].RLock()
	defer userCachem[user.ID%100].RUnlock()
	user, ok = userAccountCache[name]
	return
}

func fromID(id int) (user User, ok bool) {
	userCachem[user.ID%100].RLock()
	defer userCachem[user.ID%100].RUnlock()
	user, ok = userCache[id]
	return
}
