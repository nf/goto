// Copyright 2011 Google Inc.
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//      http://www.apache.org/licenses/LICENSE-2.0
// 
//      Unless required by applicable law or agreed to in writing, software
//      distributed under the License is distributed on an "AS IS" BASIS,
//      WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//      See the License for the specific language governing permissions and
//      limitations under the License.

package main

import (
	"bufio"
	"github.com/nf/stat"
	"gob"
	"log"
	"os"
	"rpc"
	"sync"
	"time"
)

const (
	saveTimeout     = 10e9
	saveQueueLength = 1000
)

type Store interface {
	Put(url, key *string) os.Error
	Get(key, url *string) os.Error
}


type URLStore struct {
	mu    sync.RWMutex
	urls  map[string]string
	count int
	save  chan record
}

type record struct {
	Key, URL string
}

func NewURLStore(filename string) *URLStore {
	s := &URLStore{urls: make(map[string]string)}
	if filename != "" {
		s.save = make(chan record, saveQueueLength)
		if err := s.load(filename); err != nil {
			log.Println("URLStore:", err)
		}
		go s.saveLoop(filename)
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
	if s.save != nil {
		s.save <- record{*key, *url}
	}
	return nil
}


func (s *URLStore) load(filename string) os.Error {
	f, err := os.Open(filename, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	b := bufio.NewReader(f)
	d := gob.NewDecoder(b)
	for {
		var r record
		if err := d.Decode(&r); err == os.EOF {
			break
		} else if err != nil {
			return err
		}
		if err = s.Set(&r.Key, &r.URL); err != nil {
			return err
		}
	}
	return nil
}

func (s *URLStore) saveLoop(filename string) {
	f, err := os.Open(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Println("URLStore:", err)
		return
	}
	b := bufio.NewWriter(f)
	e := gob.NewEncoder(b)
	t := time.NewTicker(saveTimeout)
	defer f.Close()
	defer b.Flush()
	for {
		var err os.Error
		select {
		case r := <-s.save:
			err = e.Encode(r)
		case <-t.C:
			err = b.Flush()
		}
		if err != nil {
			log.Println("URLStore:", err)
		}
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
	if err := s.client.Call("Store.Put", url, key); err != nil {
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
