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


type PersistentStore struct {
	mu          sync.Mutex
	urls        *UrlMap
	count       int64
	filename    string
	saveTrigger chan bool
}

func NewPersistentStore(filename string) *PersistentStore {
	s := &PersistentStore{
		urls:        NewUrlMap(),
		filename:    filename,
		saveTrigger: make(chan bool, 100), // some headroom
	}
	if err := s.load(); err != nil {
		log.Println("PersistentStore:", err)
	}
	go s.saveLoop()
	return s
}

func (s *PersistentStore) Get(key, url *string) os.Error {
	if u, ok := s.urls.Get(*key); ok {
		*url = u
		return nil
	}
	return os.NewError("key not found")
}

func (s *PersistentStore) Put(url, key *string) os.Error {
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
	s.saveTrigger <- true
	return nil
}

func (s *PersistentStore) load() os.Error {
	f, err := os.Open(s.filename, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.urls.ReadFrom(f)
}

func (s *PersistentStore) saveLoop() {
	saving := false
	timeout := make(chan bool)
	for {
		select {
		case <-s.saveTrigger:
			if !saving {
				go func() {
					time.Sleep(saveTimeout)
					timeout <- true
				}()
				saving = true
			}
		case <-timeout:
			if err := s.save(); err != nil {
				log.Println("PersistentStore:", err)
			}
			saving = false
		}
	}
}

func (s *PersistentStore) save() os.Error {
	log.Println("Saving")
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.urls.WriteTo(f)
}


type ProxyStore struct {
	urls   *UrlMap
	client *rpc.Client
}

func NewProxyStore(addr string) *ProxyStore {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Println("ProxyStore:", err)
	}
	return &ProxyStore{urls: NewUrlMap(), client: client}
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


type UrlMap struct {
	mu   sync.Mutex
	urls map[string]string
}

func NewUrlMap() *UrlMap {
	return &UrlMap{urls: make(map[string]string)}
}

func (m *UrlMap) Set(key, url string) {
	m.mu.Lock()
	m.urls[key] = url
	m.mu.Unlock()
}

func (m *UrlMap) Get(key string) (string, bool) {
	m.mu.Lock()
	url, ok := m.urls[key]
	m.mu.Unlock()
	return url, ok
}

func (m *UrlMap) WriteTo(w io.Writer) os.Error {
	e := gob.NewEncoder(w)
	m.mu.Lock()
	defer m.mu.Unlock()
	return e.Encode(m.urls)
}

func (m *UrlMap) ReadFrom(r io.Reader) os.Error {
	d := gob.NewDecoder(r)
	m.mu.Lock()
	defer m.mu.Unlock()
	return d.Decode(&m.urls)
}
