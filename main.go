package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

func main() {
	cfg := apiConfig{}
	mux := http.NewServeMux()
	//fetch files in home directory, removes the prefix /app/
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/metrics", cfg.requestCounts)
	mux.HandleFunc("/reset", cfg.reset)
	server := http.Server{Handler: mux, Addr: ":8080"}

	server.ListenAndServe()
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK"))
}

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment the counter
		cfg.fileserverHits.Add(1)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) requestCounts(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(fmt.Appendf(nil, "Hits: %v", cfg.fileserverHits.Load()))

}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}

