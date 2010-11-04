package main

import (
	"gob"
	"io"
	"log"
	"os"
	"rpc"
	"sync"
)


type Store interface {
	Put(url, key *string) os.Error
	Get(key, url *string) os.Error
}


type PersistentStore struct {
	sync.Mutex
	urls  *UrlMap
	count int64
	filename string
}

func NewPersistentStore(filename string) *PersistentStore {
	s := &PersistentStore{urls: NewUrlMap(), filename: filename}
	if err := s.load(); err != nil {
		log.Println("PersistentStore:", err)
	}
	return s
}

func (s *PersistentStore) Get(key, url *string) os.Error {
	*url, _ = s.urls.Get(*key)
	return nil
}

func (s *PersistentStore) Put(url, key *string) os.Error {
	s.Lock()
	for {
		*key = genKey(s.count)
		s.count++
		if _, ok := s.urls.Get(*key); !ok {
			break
		}
	}
	s.Unlock()
	s.urls.Set(*key, *url)
	return s.save()
}

func (s *PersistentStore) load() os.Error {
	s.Lock()
	defer s.Unlock()
	f, err := os.Open(s.filename, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.urls.ReadFrom(f)
}

func (s *PersistentStore) save() os.Error {
	s.Lock()
	defer s.Unlock()
	f, err := os.Open(s.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.urls.WriteTo(f)
}


type ProxyStore struct {
	sync.Mutex
	cache *UrlMap
	client *rpc.Client
}

func NewProxyStore(addr string) *ProxyStore {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Println("ProxyStore:", err)
	}
	return &ProxyStore{cache: NewUrlMap(), client: client}
}

func (s *ProxyStore) Get(key, url *string) os.Error {
	if u, ok := s.cache.Get(*key); ok {
		*url = u
		return nil
	}
	err := s.client.Call("PersistentStore.Get", key, url)
	if err == nil {
		s.cache.Set(*key, *url)
	}
	return err
}

func (s *ProxyStore) Put(url, key *string) os.Error {
	err := s.client.Call("PersistentStore.Put", url, key)
	if err == nil {
		s.cache.Set(*key, *url)
	}
	return err
}


type UrlMap struct {
	sync.Mutex
	urls map[string]string
}

func NewUrlMap() *UrlMap {
	return &UrlMap{urls: make(map[string]string)}
}

func (m *UrlMap) Set(key, url string) {
	m.Lock()
	m.urls[key] = url
	m.Unlock()
	log.Println("Set", key, url)
}

func (m *UrlMap) Get(key string) (string, bool) {
	m.Lock()
	url, ok := m.urls[key]
	m.Unlock()
	log.Println("Get", key, url, ok)
	return url, ok
}

func (m *UrlMap) WriteTo(w io.Writer) os.Error {
	e := gob.NewEncoder(w)
	m.Lock()
	defer m.Unlock()
	return e.Encode(m.urls)
}

func (m *UrlMap) ReadFrom(r io.Reader) os.Error {
	d := gob.NewDecoder(r)
	m.Lock()
	defer m.Unlock()
	return d.Decode(&m.urls)
}
