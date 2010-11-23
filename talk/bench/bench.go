package main

import (
	"flag"
	"fmt"
	"github.com/nf/stat"
	"http"
	"io/ioutil"
	"log"
	"rand"
	"time"
)

var (
	host = flag.String("host", "localhost:8080", "target host:port")
)

const (
	fooUrl = "http://example.net/foobar"
	monDelay  = 1e9
	getDelay  = 50e6
	postDelay = 50e6
	getters   = 100
	posters   = 10
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
			urls = append(urls, u)
		case randURL <- r:
		}
	}
}

func post() {
	url := fmt.Sprintf("http://%s/add", *host)
	r, err := http.PostForm(url, map[string]string{"url": fooUrl})
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
	stat.In <- "post"
}

func get() {
	url := <-randURL
	r, err := http.Head(url)
	if err != nil {
		log.Println("get:", err)
		return
	}
	defer r.Body.Close()
	if r.StatusCode != 302 {
		log.Println("get: wrong StatusCode:", r.StatusCode)
	}
	if l := r.Header["Location"]; l != fooUrl {
		log.Println("get: wrong Location:", l)
	}
	stat.In <- "get"
}

func loop(fn func(), delay int64) {
	for {
		fn()
		time.Sleep(getDelay)
	}
}

func main() {
	flag.Parse()
	rand.Seed(time.Nanoseconds())
	go keeper()
	for i := 0; i < getters; i++ {
		go loop(get, getDelay)
	}
	for i := 0; i < posters; i++ {
		go loop(post, postDelay)
	}
	stat.Monitor(monDelay)
}
