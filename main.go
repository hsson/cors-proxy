package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

const corsHeaderKey = "Access-Control-Allow-Origin"

var client = http.DefaultClient

var (
	port = flag.Int("port", 0, "override default port (80 for http, 443 for https)")

	certPath = flag.String("tls-cert", "", "specify path to TLS .crt/.pem public certificate (default \"\")")
	keyPath  = flag.String("tls-key", "", "specify path to the TLS private key (default \"\")")
	useTLS   = flag.Bool("tls", false, "set to true to serve using TLS (default: false)")

	verbose = flag.Bool("v", false, "verbose flag, log all incoming requests (default: false)")
)

func main() {
	flag.Parse()

	router := mux.NewRouter()
	router.HandleFunc("/proxy", proxy)

	fmt.Println("starting CORS proxy")

	if *useTLS {
		p := 443
		if *port != 0 {
			p = *port
		}
		if err := http.ListenAndServeTLS(fmt.Sprintf(":%d", p), *certPath, *keyPath, router); err != nil {
			log.Fatalf("error: %v", err)
		}
	} else {
		p := 80
		if *port != 0 {
			p = *port
		}
		if err := http.ListenAndServe(fmt.Sprintf(":%d", p), router); err != nil {
			log.Fatalf("error: %v", err)
		}
	}
}

func proxy(w http.ResponseWriter, r *http.Request) {
	proxyURL := r.URL.Query().Get("u")
	if *verbose {
		log.Printf("proxying: %v", proxyURL)
	}

	request, err := http.NewRequest(r.Method, proxyURL, r.Body)
	if err != nil {
		http.Error(w, "cors-proxy: could not create request to "+proxyURL+"\n"+err.Error(), http.StatusBadRequest)
		return
	}

	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "hsson/cors-proxy"
	}
	request.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(request)
	if err != nil {
		http.Error(w, "cors-proxy: request error\n"+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set(corsHeaderKey, "*")
	for k, v := range resp.Header {
		if k == corsHeaderKey {
			continue
		}
		for _, s := range v {
			w.Header().Add(k, s)
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "cors-proxy: failed to read body\n"+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(body); err != nil {
		log.Printf("write body failed: %v", err)
	}
}
