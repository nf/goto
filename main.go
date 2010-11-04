package main

import (
	"flag"
	"fmt"
	"http"
	"rpc"
)

var (
	listenAddr = flag.String("http", ":9980", "http listen address")
	dataFile   = flag.String("file", "store.gob", "data store file name")
	hostname   = flag.String("host", "r.nf.id.au", "http host name")
	password   = flag.String("pass", "", "password")
	masterAddr = flag.String("master", "", "RPC master address")
	rpcEnabled = flag.Bool("rpc", false, "act as RPC master server")
)

var store Store

func main() {
	flag.Parse()
	if *masterAddr != "" {
		store = NewProxyStore(*masterAddr)
	} else {
		store = NewPersistentStore(*dataFile)
	}
	if *rpcEnabled {
		rpc.Register(store)
		rpc.HandleHTTP()
	}
	http.HandleFunc("/", Redirect)
	http.HandleFunc("/add", Add)
	http.ListenAndServe(*listenAddr, nil)
}

func Redirect(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[1:]
	var url string
	if err := store.Get(&key, &url); err != nil {
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	if url == "" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func Add(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if url == "" {
		fmt.Fprint(w, addform)
		return
	}
	var key string
	if err := store.Put(&url, &key); err != nil { 
		http.Error(w, err.String(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "http://%s/%s", *hostname, key)
}

const addform = `
<form method="POST" action="/add">
URL: <input type="text" name="url">
<input type="submit" value="Add">
</form>
`
