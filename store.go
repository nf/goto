package main

import (
	"gob"
	"io"
	"os"
	"sync"
)

type Store struct {
	urls  map[string]string
	count int64
	mu    sync.Mutex
}

func NewStore() *Store {
	return &Store{urls: make(map[string]string)}
}

func (s *Store) Get(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.urls[key]
}

func (s *Store) Set(target string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var key string
	for {
		key = base36(s.count)
		s.count++
		if _, ok := s.urls[key]; !ok {
			break
		}
	}
	s.urls[key] = target
	return key
}

func (s *Store) SetCustom(key, target string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls[key] = target
}

func (s *Store) WriteTo(w io.Writer) os.Error {
	e := gob.NewEncoder(w)
	s.mu.Lock()
	defer s.mu.Unlock()
	return e.Encode(s)
}

func (s *Store) ReadFrom(r io.Reader) os.Error {
	d := gob.NewDecoder(r)
	s.mu.Lock()
	defer s.mu.Unlock()
	return d.Decode(s)
}

var base36Char = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func base36(n int64) string {
	if n == 0 {
		return string(base36Char[0])
	}
	s := make([]byte, 20)
	i := len(s)
	for n > 0 && i >= 0 {
		i--
		j := n % 36
		n = (n - j) / 36
		s[i] = base36Char[j]
	}
	return string(s[i:])
}
