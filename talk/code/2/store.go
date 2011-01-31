package main

import (
	"gob"
	"log"
	"os"
	"sync"
)

const saveQueueLength = 1000

type URLStore struct {
	urls  map[string]string
	mu    sync.RWMutex
	count int
	save  chan record
}

type record struct {
	Key, URL string
}

func NewURLStore(filename string) *URLStore {
	s := &URLStore{
		urls: make(map[string]string),
		save: make(chan record, saveQueueLength),
	}
	if err := s.load(filename); err != nil {
		log.Println("URLStore:", err)
	}
	go s.saveLoop(filename)
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
			s.save <- record{key, url}
			return key
		}
	}
	panic("shouldn't get here")
}

func (s *URLStore) load(filename string) os.Error {
	f, err := os.Open(filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	d := gob.NewDecoder(f)
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

func (s *URLStore) saveLoop(filename string) {
	f, err := os.Open(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Println("URLStore:", err)
		return
	}
	e := gob.NewEncoder(f)
	for {
		r := <-s.save
		if err := e.Encode(r); err != nil {
			log.Println("URLStore:", err)
		}
	}
}
