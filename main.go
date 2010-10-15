package main

import (
	"flag"
	"fmt"
	"http"
	"log"
	"os"
	"sync"
)

var (
	listenAddr = flag.String("http", ":9980", "http listen address")
	dataFile   = flag.String("file", "store.gob", "data store file name")
	hostname   = flag.String("host", "r.nf.id.au", "http host name")
	password   = flag.String("pass", "", "password")
)

func main() {
	flag.Parse()
	loadStore()
	http.HandleFunc("/", Redirect)
	http.HandleFunc("/add", Add)
	http.ListenAndServe(*listenAddr, nil)
}

var (
	store     = NewStore()
	storeLock sync.Mutex
)

func Redirect(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[1:]
	target := store.Get(key)
	if target == "" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func Add(w http.ResponseWriter, r *http.Request) {
	key, target := r.FormValue("key"), r.FormValue("target")
	if key == "" && target == "" {
		fmt.Fprint(w, addform)
		return
	}
	pw := r.FormValue("pw")
	if pw != *password {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if key == "" {
		key = store.Set(target)
	} else {
		store.SetCustom(key, target)
	}
	fmt.Fprintf(w, "http://%s/%s", *hostname, key)
	saveStore()
}

func loadStore() {
	storeLock.Lock()
	defer storeLock.Unlock()
	f, err := os.Open(*dataFile, os.O_RDONLY, 0644)
	if err != nil {
		log.Printf("error opening %q: %s", *dataFile, err)
		return
	}
	defer f.Close()
	if err := store.ReadFrom(f); err != nil {
		log.Printf("error reading %q: %s", *dataFile, err)
	}
}

func saveStore() {
	storeLock.Lock()
	defer storeLock.Unlock()
	f, err := os.Open(*dataFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("error opening %q: %s", *dataFile, err)
		return
	}
	defer f.Close()
	if err := store.WriteTo(f); err != nil {
		log.Printf("error writing %q: %s", *dataFile, err)
	}
}

const addform = `
<form method="POST" action="/add">
<input type="password" name="pw" length="10">
<input type="text" name="target">
<input type="text" name="key">
<input type="submit" value="Add">
</form>
`
