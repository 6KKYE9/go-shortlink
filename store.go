package main

import (
	"errors"
	"sync"
)

// ErrNotFound 短码不在库里。
var ErrNotFound = errors.New("short code not found")

// Store 是短链接的存储抽象。真上分布式可以换成 Redis/MySQL 实现，
// 服务层只认这个接口，不关心底层是单机的还是集群的。
type Store interface {
	// Save 把短码和原始 URL 存起来，短码已存在就返回 error。
	Save(code, url string) error
	// Load 按短码取原始 URL，没有就返回 ErrNotFound。
	Load(code string) (string, error)
}

// memStore 是练手用的内存版实现，带读写锁。
type memStore struct {
	mu sync.RWMutex
	m  map[string]string
}

func newMemStore() *memStore {
	return &memStore{m: make(map[string]string)}
}

func (s *memStore) Save(code, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[code]; ok {
		return errors.New("code already exists")
	}
	s.m[code] = url
	return nil
}

func (s *memStore) Load(code string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.m[code]
	if !ok {
		return "", ErrNotFound
	}
	return url, nil
}
