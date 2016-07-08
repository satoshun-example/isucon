package main

import (
	"sync"
	"time"
)

type lastLogin struct {
	id        int
	login     string
	ip        string
	CreatedAt time.Time
}

type MultiMapLastLogin struct {
	lock []sync.RWMutex
	data []map[int]lastLogin
	size int
}

func NewLoginMap(size int) *MultiMapLastLogin {
	if size <= 0 {
		size = 256
	}

	lock := make([]sync.RWMutex, size)
	data := make([]map[int]lastLogin, size)
	for i := 0; i < size; i++ {
		lock[i] = sync.RWMutex{}
		data[i] = make(map[int]lastLogin)
	}

	return &MultiMapLastLogin{
		lock: lock,
		data: data,
		size: size,
	}
}

func (m *MultiMapLastLogin) Del(key int) {
	k := hash(key, m.size)

	m.lock[k].Lock()
	delete(m.data[k], key)
	m.lock[k].Unlock()
}

func (m *MultiMapLastLogin) Set(key int, value lastLogin) {
	k := hash(key, m.size)

	m.lock[k].Lock()
	m.data[k][key] = value
	m.lock[k].Unlock()
}

func (m *MultiMapLastLogin) Get(key int) (lastLogin, bool) {
	k := hash(key, m.size)

	m.lock[k].RLock()
	defer m.lock[k].RUnlock()

	v, ok := m.data[k][key]
	return v, ok
}

func (m *MultiMapLastLogin) Has(key int) bool {
	k := hash(key, m.size)

	m.lock[k].RLock()
	defer m.lock[k].RUnlock()

	_, ok := m.data[k][key]
	return ok
}

// func (m *MultiMapLastLogin) Incr(key int) {
// 	k := hash(key, m.size)

// 	m.lock[k].Lock()
// 	defer m.lock[k].Unlock()

// 	v, ok := m.data[k][key]
// 	if ok {
// 		m.data[k][key] = v + 1
// 	} else {
// 		m.data[k][key] = 1
// 	}
// }

func hash(key, size int) int {
	return key % size
}
