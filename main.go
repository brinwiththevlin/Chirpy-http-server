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
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("GET /admin/metrics", cfg.requestCounts)
	mux.HandleFunc("POST /admin/reset", cfg.reset)
	server := http.Server{Handler: mux, Addr: ":8080"}

	server.ListenAndServe()
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK\n"))
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(fmt.Appendf(nil, "<html> <body> <h1>Welcome, Chirpy Admin</h1> <p>Chirpy has been visited %d times!</p> </body> </html>", cfg.fileserverHits.Load()))

}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}
