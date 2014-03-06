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
	"flag"
	"fmt"
	"github.com/nf/stat"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	n          = flag.Int("n", 10, "magnitude of assault")
	host       = flag.String("host", "localhost:8080", "target host:port")
	statServer = flag.String("stats", "localhost:8090", "stat server host")
	hosts      []string
	hostRe     = regexp.MustCompile("http://[a-zA-Z0-9:.]+")
)

const (
	fooUrl    = "http://example.net/foobar"
	monDelay  = 1e9
	getDelay  = 100e6
	getters   = 10
	postDelay = 100e6
	posters   = 1
)

var (
	newURL  = make(chan string)
	randURL = make(chan string)
)

func keeper() {
	var urls []string
	urls = append(urls, <-newURL)
	for {
		r := urls[rand.Intn(len(urls))]
		select {
		case u := <-newURL:
			for _, h := range hosts {
				u = hostRe.ReplaceAllString(u, "http://"+h)
				urls = append(urls, u)
			}
		case randURL <- r:
		}
	}
}

func post() {
	u := fmt.Sprintf("http://%s/add", hosts[rand.Intn(len(hosts))])
	r, err := http.PostForm(u, url.Values{"url": {fooUrl}})
	if err != nil {
		log.Println("post:", err)
		return
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("post:", err)
		return
	}
	newURL <- string(b)
	stat.In <- "put"
}

func get() {
	u := <-randURL
	req, err := http.NewRequest("HEAD",u,nil)
	r, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Println("get:", err)
		return
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("get:", err)
		return
	}
	if r.StatusCode != 302 {
		log.Println("get: wrong StatusCode:", r.StatusCode)
		if r.StatusCode == 500 {
			log.Printf("Error: %s\n", b)
		}
	}
	if l := r.Header.Get("Location"); l != fooUrl {
		log.Println("get: wrong Location:", l)
	}
	stat.In <- "get"
}

func loop(fn func(), delay time.Duration) {
	for {
		fn()
		time.Sleep(delay)
	}
}

func main() {
	flag.Parse()
	hosts = strings.Split(*host, ",")
	rand.Seed(time.Now().UnixNano())
	go keeper()
	for i := 0; i < getters*(*n); i++ {
		go loop(get, getDelay)
	}
	for i := 0; i < posters*(*n); i++ {
		go loop(post, postDelay)
	}
	stat.Process = "!bench"
	stat.Monitor(*statServer)
}
