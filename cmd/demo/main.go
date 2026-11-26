package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	workRequests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "demo_work_requests_total",
		Help: "Number of /work responses by version and HTTP status.",
	}, []string{"version", "status"})
	prometheus.MustRegister(workRequests)

	workDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "demo_work_duration_seconds",
		Help: "Time spent handling /work requests.",
	}, []string{"version"})
	prometheus.MustRegister(workDuration)
	
	mux.Handle("/metrics", promhttp.Handler())
	
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, version)
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request){
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/work", func(w http.ResponseWriter, r *http.Request){
		started := time.Now()
		defer func() {
			workDuration.WithLabelValues(version).Observe(time.Since(started).Seconds())
		}()
		time.Sleep(delay)
		if demo_fail {
			http.Error(w, "demo failure", http.StatusInternalServerError)
			workRequests.WithLabelValues(version, "500").Inc()
			return
		}
		fmt.Fprintln(w, "work complete")
		workRequests.WithLabelValues(version, "200").Inc()
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
