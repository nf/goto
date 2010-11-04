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
	Put(key string) (string, os.Error)
	Get(key string) (string, os.Error)
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

func (s *PersistentStore) Get(key string) (string, os.Error) {
	url, _ := s.urls.Get(key)
	return url, nil
}

func (s *PersistentStore) Put(url string) (string, os.Error) {
	var key string
	s.Lock()
	for {
		key = genKey(s.count)
		s.count++
		if _, ok := s.urls.Get(key); !ok {
			break
		}
	}
	s.Unlock()
	s.urls.Set(key, url)
	err := s.save()
	return key, err
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
	p := &ProxyStore{cache: NewUrlMap()}
	// TODO: create client connection
	return p
}

func (s *ProxyStore) Get(key string) (string, os.Error) {
	if url, ok := s.cache.Get(key); ok {
		return url, nil
	}
	var url string
	err := s.client.Call("Store.Get", &key, &url)
	if err == nil {
		s.cache.Set(key, url)
	}
	return url, err
}

func (s *ProxyStore) Put(key string) (string, os.Error) {
	var url string
	err := s.client.Call("Store.Put", &key, &url)
	if err == nil {
		s.cache.Set(key, url)
	}
	return url, err
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
}

func (m *UrlMap) Get(key string) (string, bool) {
	m.Lock()
	defer m.Unlock()
	url, ok := m.urls[key]
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


var keyChar = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func genKey(n int64) string {
	if n == 0 {
		return string(keyChar[0])
	}
	l := int64(len(keyChar))
	s := make([]byte, 20) // FIXME: will overflow. eventually.
	i := len(s)
	for n > 0 && i >= 0 {
		i--
		j := n % l
		n = (n - j) / l
		s[i] = keyChar[j]
	}
	return string(s[i:])
}
