package main

import (
	"gob"
	"io"
	"log"
	"os"
	"rpc"
	"sync"
	"time"
)

const saveTimeout = 60e9

type Store interface {
	Put(url, key *string) os.Error
	Get(key, url *string) os.Error
}

type URLStore struct {
	mu       sync.Mutex
	urls     *URLMap
	count    int
	filename string
	dirty    chan bool
}

func NewURLStore(filename string) *URLStore {
	s := &URLStore{
		urls:     NewURLMap(),
		filename: filename,
		dirty:    make(chan bool, 1),
	}
	if err := s.load(); err != nil {
		log.Println("URLStore:", err)
	}
	go s.saveLoop()
	return s
}

func (s *URLStore) Get(key, url *string) os.Error {
	log.Println("URLStore: Get", *key)
	if u := s.urls.Get(*key); u != "" {
		*url = u
		return nil
	}
	return os.NewError("key not found")
}

func (s *URLStore) Put(url, key *string) os.Error {
	log.Println("URLStore: Put", *url)
	s.mu.Lock()
	for {
		*key = genKey(s.count)
		s.count++
		if u := s.urls.Get(*key); u == "" {
			break
		}
	}
	s.urls.Set(*key, *url)
	s.mu.Unlock()
	_ = s.dirty <- true
	return nil
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

func (s *URLStore) saveLoop() {
	for {
		<-s.dirty
		log.Println("URLStore: saving")
		if err := s.save(); err != nil {
			log.Println("URLStore:", err)
		}
		time.Sleep(saveTimeout)
	}
}


type ProxyStore struct {
	urls   *URLMap
	client *rpc.Client
}

func NewProxyStore(addr string) *ProxyStore {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Println("ProxyStore:", err)
	}
	return &ProxyStore{urls: NewURLMap(), client: client}
}

func (s *ProxyStore) Get(key, url *string) os.Error {
	if u := s.urls.Get(*key); u != "" {
		*url = u
		log.Println("ProxyStore: Get cache hit", *key)
		return nil
	}
	log.Println("ProxyStore: Get cache miss", *key)
	err := s.client.Call("Store.Get", key, url)
	if err == nil {
		s.urls.Set(*key, *url)
	}
	return err
}

func (s *ProxyStore) Put(url, key *string) os.Error {
	log.Println("ProxyStore: Put", *url)
	err := s.client.Call("Store.Put", url, key)
	if err == nil {
		s.urls.Set(*key, *url)
	}
	return err
}


type URLMap struct {
	mu   sync.RWMutex
	urls map[string]string
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
