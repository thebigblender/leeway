package main

import (
	"fmt"
	"log"
	"time"
	"net/http"
	"os"
)

func main() {
	version := os.Getenv("APP_VERSION")
	demo_fail := os.Getenv("DEMO_FAIL") == "1"

	var delay time.Duration
	if rawDelay := os.Getenv("DEMO_DELAY"); rawDelay != "" {
		parsedDelay, err := time.ParseDuration(rawDelay)
		if err != nil || parsedDelay < 0 {
			log.Fatalf("invalid DEMO_DELAY %q", rawDelay)
		}
		delay = parsedDelay
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, version)
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request){
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/work", func(w http.ResponseWriter, r *http.Request){
		time.Sleep(delay)
		if demo_fail {
			http.Error(w, "demo failure", http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, "work complete")
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request){
		time.Sleep(5 * time.Second)
		fmt.Fprintln(w, "slow work complete")
	})

	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8000"
	}
	log.Fatal(http.ListenAndServe(addr, mux))
}
