package main

import (
	"flag"
	"fmt"
	"github.com/nf/stat"
	"http"
	"io/ioutil"
	"log"
	"rand"
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
	url := fmt.Sprintf("http://%s/add", hosts[rand.Intn(len(hosts))])
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
	stat.In <- "put"
}

func get() {
	url := <-randURL
	r, err := http.Head(url)
	if err != nil {
		log.Println("get:", err)
		return
	}
	defer r.Body.Close()
	if _, err := ioutil.ReadAll(r.Body); err != nil {
		log.Println("get:", err)
		return
	}
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
	hosts = strings.Split(*host, ",", -1)
	rand.Seed(time.Nanoseconds())
	go keeper()
	for i := 0; i < getters*(*n); i++ {
		go loop(get, getDelay)
	}
	for i := 0; i < posters*(*n); i++ {
		go loop(post, postDelay)
	}
	stat.Process = "bench"
	stat.Monitor(*statServer)
}
