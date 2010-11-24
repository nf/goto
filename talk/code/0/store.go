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
		if u := s.urls.Get(key); u == "" {
			break
		}
	}
	s.urls.Set(key, url)
	s.mu.Unlock()
	return
}

func (s *URLStore) Get(key string) (url string) {
	return s.urls.Get(key)
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

func (m *URLMap) Get(key string) (url string) {
	m.mu.RLock()
	url = m.urls[key]
	m.mu.RUnlock()
	return
}
