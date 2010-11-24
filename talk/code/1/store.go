package main

import (
	"gob"
	"io"
	"log"
	"os"
	"sync"
)

type URLStore struct {
	mu       sync.Mutex
	urls     *URLMap
	count    int
	filename string
}

func NewURLStore(filename string) *URLStore {
	s := &URLStore{
		urls:     NewURLMap(),
		filename: filename,
	}
	if err := s.load(); err != nil {
		log.Println("URLStore:", err)
	}
	return s
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
	if err := s.save(); err != nil {
		log.Println("URLStore:", err)
	}
	s.mu.Unlock()
	return
}

func (s *URLStore) Get(key string) (url string) {
	if u, ok := s.urls.Get(key); ok {
		url = u
	}
	return
}

func (s *URLStore) load() os.Error {
	f, err := os.Open(s.filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.urls.ReadFrom(f)
}

func (s *URLStore) save() os.Error {
	f, err := os.Open(s.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.urls.WriteTo(f)
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

func (m *URLMap) WriteTo(w io.Writer) os.Error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e := gob.NewEncoder(w)
	return e.Encode(m.urls)
}

func (m *URLMap) ReadFrom(r io.Reader) os.Error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d := gob.NewDecoder(r)
	return d.Decode(&m.urls)
}
