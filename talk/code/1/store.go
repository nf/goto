package main

import (
	"gob"
	"log"
	"os"
	"sync"
)

type URLStore struct {
	urls     map[string]string
	mu       sync.RWMutex
	count    int
	file     *os.File
}

type record struct {
	Key, URL string
}

func NewURLStore(filename string) *URLStore {
	s := &URLStore{urls: make(map[string]string)}
	f, err := os.Open(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Println("URLStore:", err)
		return s
	}
	s.file = f
	if err := s.load(); err != nil {
		log.Println("URLStore:", err)
	}
	return s
}

func (s *URLStore) Get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.urls[key]
}

func (s *URLStore) Set(key, url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, present := s.urls[key]; present {
		return false
	}
	s.urls[key] = url
	return true
}

func (s *URLStore) Put(url string) string {
	for {
		key := genKey(s.count)
		s.count++
		if ok := s.Set(key, url); ok {
			if err := s.save(key, url); err != nil {
				log.Println("URLStore:", err)
			}
			return key
		}
	}
	panic("shouldn't get here")
}

func (s *URLStore) load() os.Error {
	if _, err := s.file.Seek(0, 0); err != nil {
		return err
	}
	d := gob.NewDecoder(s.file)
	for {
		var r record
		if err := d.Decode(&r); err == os.EOF {
			break
		} else if err != nil {
			return err
		}
		s.Set(r.Key, r.URL)
	}
	return nil
}

func (s *URLStore) save(key, url string) os.Error {
	e := gob.NewEncoder(s.file)
	return e.Encode(record{key, url})
}
