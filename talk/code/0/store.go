package main

import "sync"

type URLStore struct {
	mu    sync.Mutex
	urls  *URLMap
	count int
}

func NewURLStore() *URLStore {
	return &URLStore{urls: NewURLMap()}
}

func (s *URLStore) Put(url string) (key string) {
	s.mu.Lock()
	for {
		key = genKey(s.count)
		s.count++
		if _, ok := s.urls.Get(key); !ok {
			break
		}
	}
	s.urls.Set(key, url)
	s.mu.Unlock()
	return
}

func (s *URLStore) Get(key string) (url string) {
	if u, ok := s.urls.Get(key); ok {
		url = u
	}
	return
}


type URLMap struct {
	urls map[string]string
	mu   sync.RWMutex
}

func NewURLMap() *URLMap {
	return &URLMap{urls: make(map[string]string)}
}

func (m *URLMap) Set(key, url string) {
	m.mu.Lock()
	m.urls[key] = url
	m.mu.Unlock()
}

func (m *URLMap) Get(key string) (string, bool) {
	m.mu.RLock()
	url, ok := m.urls[key]
	m.mu.RUnlock()
	return url, ok
}
