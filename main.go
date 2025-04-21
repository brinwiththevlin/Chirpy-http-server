package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/brinwiththevlin/Chirpy-http-server/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)
	cfg := apiConfig{db: dbQueries, platform: platform}
	mux := http.NewServeMux()
	//fetch files in home directory, removes the prefix /app/
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", cfg.requestCounts)
	mux.HandleFunc("POST /admin/reset", cfg.reset)
	mux.HandleFunc("POST /api/users", cfg.usersHandler)
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("POST /api/validate_chirp", validHandler)

	server := http.Server{Handler: mux, Addr: ":8080"}

	server.ListenAndServe()
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK\n"))
}

func validHandler(w http.ResponseWriter, r *http.Request) {
	type reqBody struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	body := reqBody{}
	err := decoder.Decode(&body)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	if len(body.Body) >= 140 {
		w.WriteHeader(400)
		return
	}
	profane := []string{"kerfuffle", "sharbert", "fornax"}
	words := []string{}
	for _, word := range strings.Fields(body.Body) {
		if slices.Contains(profane, strings.ToLower(word)) {
			words = append(words, "****")
		} else {
			words = append(words, word)
		}
	}
	chirp := strings.Join(words, " ")

	type respBody struct {
		Body string `json:"cleaned_body"`
	}
	retBody := respBody{Body: chirp}
	dat, err := json.Marshal(&retBody)
	if err != nil {
		log.Printf("Error marshaling: %v\n", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)

}

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
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
	w.Write(fmt.Appendf(nil, "<html> <body> <h1>Welcome, Chirpy Admin</h1> <p>Chirpy has been visited %d times!</p> </body> </html>",
		cfg.fileserverHits.Load()))

}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		return
	}
	cfg.fileserverHits.Store(0)
	err := cfg.db.RemoveUsers(r.Context())
	if err != nil {
		log.Println("could not remove all users")
		w.WriteHeader(500)
		return
	}

}

func (cfg *apiConfig) usersHandler(w http.ResponseWriter, r *http.Request) {
	type reqBody struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	body := reqBody{}
	err := decoder.Decode(&body)

	if err != nil {
		w.WriteHeader(400)
		return
	}
	user, err := cfg.db.CreateUser(r.Context(), body.Email)
	if err != nil {
		log.Printf("user email is in use: %v", body.Email)
		w.WriteHeader(500)
		return
	}

	type respBody struct {
		ID        string `json:"id"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Email     string `json:"email"`
	}

	res := respBody{
		ID:        user.ID.String(),
		CreatedAt: user.CreatedAt.Local().Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Local().Format(time.RFC3339),
		Email:     user.Email,
	}

	dat, err := json.Marshal(&res)
	if err != nil {
		log.Printf("Error marshaling: %v\n", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(dat)

}
