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

const saveTimeout = 10e9

type Store interface {
	Put(url, key *string) os.Error
	Get(key, url *string) os.Error
}


type URLStore struct {
	mu       sync.Mutex
	urls     *URLMap
	count    int64
	filename string
	dirtied  chan bool
}

func NewURLStore(filename string) *URLStore {
	s := &URLStore{
		urls:        NewURLMap(),
		filename:    filename,
		saveTrigger: make(chan bool, 1000), // some headroom
	}
	if err := s.load(); err != nil {
		log.Println("URLStore:", err)
	}
	go s.saveLoop()
	return s
}

func (s *URLStore) Get(key, url *string) os.Error {
	if u, ok := s.urls.Get(*key); ok {
		*url = u
		return nil
	}
	return os.NewError("key not found")
}

func (s *URLStore) Put(url, key *string) os.Error {
	s.mu.Lock()
	for {
		*key = genKey(s.count)
		s.count++
		if _, ok := s.urls.Get(*key); !ok {
			break
		}
	}
	s.mu.Unlock()
	s.urls.Set(*key, *url)
	s.dirtied <- true
	return nil
}

func (s *URLStore) load() os.Error {
	f, err := os.Open(s.filename, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.urls.ReadFrom(f)
}

func (s *URLStore) save() os.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.urls.WriteTo(f)
}

func (s *URLStore) saveLoop() {
	saving := false
	timeout := make(chan bool)
	for {
		select {
		case <-s.dirtied:
			if !saving {
				go func() {
					time.Sleep(saveTimeout)
					timeout <- true
				}()
				saving = true
			}
		case <-timeout:
			if err := s.save(); err != nil {
				log.Println("URLStore:", err)
			}
			saving = false
		}
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
	if u, ok := s.urls.Get(*key); ok {
		*url = u
		return nil
	}
	err := s.client.Call("Store.Get", key, url)
	if err == nil {
		s.urls.Set(*key, *url)
	}
	return err
}

func (s *ProxyStore) Put(url, key *string) os.Error {
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

func (m *URLMap) Get(key string) (string, bool) {
	m.mu.RLock()
	url, ok := m.urls[key]
	m.mu.RUnlock()
	return url, ok
}

func (m *URLMap) WriteTo(w io.Writer) os.Error {
	e := gob.NewEncoder(w)
	m.mu.RLock()
	defer m.mu.RUnlock()
	return e.Encode(m.urls)
}

func (m *URLMap) ReadFrom(r io.Reader) os.Error {
	d := gob.NewDecoder(r)
	m.mu.Lock()
	defer m.mu.Unlock()
	return d.Decode(&m.urls)
}
