package main

import (
	"github.com/nf/stat"
	"gob"
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
	mu       sync.RWMutex
	urls     map[string]string
	count    int
	filename string
	dirty    chan bool
}

func NewURLStore(filename string) *URLStore {
	s := &URLStore{
		urls:     make(map[string]string),
		filename: filename,
		dirty:    make(chan bool, 1),
	}
	if filename != "" {
		if err := s.load(); err != nil {
			log.Println("URLStore:", err)
		}
		go s.saveLoop()
	}
	return s
}

func (s *URLStore) Get(key, url *string) os.Error {
	defer statSend("store get")
	s.mu.RLock()
	defer s.mu.RUnlock()
	if u, ok := s.urls[*key]; ok {
		*url = u
		return nil
	}
	return os.NewError("key not found")
}

func (s *URLStore) Set(key, url *string) os.Error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, present := s.urls[*key]; present {
		return os.NewError("key already exists")
	}
	s.urls[*key] = *url
	return nil
}

func (s *URLStore) Put(url, key *string) os.Error {
	defer statSend("store put")
	for {
		*key = genKey(s.count)
		s.count++
		if err := s.Set(key, url); err == nil {
			break
		}
	}
	if s.filename != "" {
		_ = s.dirty <- true
	}
	return nil
}


func (s *URLStore) load() os.Error {
	f, err := os.Open(s.filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	d := gob.NewDecoder(f)
	return d.Decode(&s.urls)
}

func (s *URLStore) save() os.Error {
	f, err := os.Open(s.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	s.mu.RLock()
	defer s.mu.RUnlock()
	e := gob.NewEncoder(f)
	return e.Encode(s.urls)
}

func (s *URLStore) saveLoop() {
	for {
		<-s.dirty
		if err := s.save(); err != nil {
			log.Println("URLStore:", err)
		}
		time.Sleep(saveTimeout)
	}
}


type ProxyStore struct {
	urls   *URLStore
	client *rpc.Client
}

func NewProxyStore(addr string) *ProxyStore {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Println("ProxyStore:", err)
	}
	return &ProxyStore{urls: NewURLStore(""), client: client}
}

func (s *ProxyStore) Get(key, url *string) os.Error {
	if err := s.urls.Get(key, url); err == nil {
		return nil
	}
	if err := s.client.Call("Store.Get", key, url); err != nil {
		return err
	}
	s.urls.Set(key, url)
	return nil
}

func (s *ProxyStore) Put(url, key *string) os.Error {
	if err := s.client.Call("Store.Put", url, key); err != nil{
		return err
	}
	s.urls.Set(key, url)
	return nil
}


func statSend(s string) {
	if *statServer != "" {
		stat.In <- s
	}
}
