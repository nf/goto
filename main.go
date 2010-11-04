package main

import (
	"flag"
	"fmt"
	"http"
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
	store = NewPersistentStore(*dataFile)
	http.HandleFunc("/", Redirect)
	http.HandleFunc("/add", Add)
	http.ListenAndServe(*listenAddr, nil)
}

var (
	store     Store
	storeLock sync.Mutex
)

func Redirect(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[1:]
	target, err := store.Get(key)
	if err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	if target == "" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func Add(w http.ResponseWriter, r *http.Request) {
	target := r.FormValue("target")
	if target == "" {
		fmt.Fprint(w, addform)
		return
	}
	if r.FormValue("pw") != *password {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	key, err := store.Put(target)
	if err != nil { 
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "http://%s/%s", *hostname, key)
}

const addform = `
<form method="POST" action="/add">
<input type="password" name="pw" length="10">
<input type="text" name="target">
<input type="submit" value="Add">
</form>
`
