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

	"github.com/brinwiththevlin/Chirpy-http-server/internal/auth"
	"github.com/brinwiththevlin/Chirpy-http-server/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	secret := os.Getenv("SECRET")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)
	cfg := apiConfig{db: dbQueries, platform: platform, secret: secret}
	mux := http.NewServeMux()
	//fetch files in home directory, removes the prefix /app/
	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("GET /admin/metrics", cfg.requestCounts)
	mux.HandleFunc("POST /admin/reset", cfg.reset)
	mux.HandleFunc("POST /api/users", cfg.usersHandler)
	mux.HandleFunc("POST /api/chirps", cfg.postHandler)
	mux.HandleFunc("GET /api/chirps", cfg.chirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.singleChirpHander)
	mux.HandleFunc("POST /api/login", cfg.loginHandler)

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
	db             *database.Queries
	platform       string
	secret         string
}

func (cfg *apiConfig) singleChirpHander(w http.ResponseWriter, r *http.Request) {
	chirpid := r.PathValue("chirpID")
	if chirpid == "" {
		w.WriteHeader(404)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("chirpID not passed"))
		log.Println("chirpID not passed")
		return
	}
	uid, err := uuid.Parse(chirpid)
	if err != nil {
		w.WriteHeader(400)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("invalid UUID"))
		log.Println("invalid UUID:", err)
	}
	chirp, err := cfg.db.GetChirp(r.Context(), uid)
	if err != nil {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("chirpID not found"))
		log.Println("chirpID not found")
		return
	}
	dat, err := json.Marshal(&chirp)
	if err != nil {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("unable to marshal"))
		log.Println("unable to marshal")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)

}

func (cfg *apiConfig) chirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("could not retrieve Chirps"))
		log.Println("could not retrieve Chirps")
		return
	}
	type chirpsArray struct {
		chirps []database.Chirp
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	dat, err := json.Marshal(&chirps)
	if err != nil {
		log.Printf("Error marshaling: %v\n", err)
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("Error marshaling"))
		return
	}

	w.Write(dat)
}

func (cfg *apiConfig) postHandler(w http.ResponseWriter, r *http.Request) {
	type reqBody struct {
		Body string `json:"body"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(401)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("unauthorized"))
		return
	}
	uid, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("could not validate JWT"))
		return
	}
	decoder := json.NewDecoder(r.Body)
	body := reqBody{}
	err = decoder.Decode(&body)
	if err != nil {
		w.WriteHeader(400)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("request body could not be decoded"))
		log.Println("request body could not be decoded")
		return
	}

	// if body.UserID != uid {
	// 	w.WriteHeader(401)
	// 	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// 	w.Write([]byte("invalid user token"))
	// 	return
	//
	// }

	if len(body.Body) >= 140 {
		w.WriteHeader(400)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("body is too long"))
		log.Println("body is too long")
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
	msg := strings.Join(words, " ")

	chirp := database.CreateChirpParams{Body: msg, UserID: uid}
	chirpRow, err := cfg.db.CreateChirp(r.Context(), chirp)
	if err != nil {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("could not log the chirp"))
		log.Println("could not log the chirp :(")
		return
	}

	dat, err := json.Marshal(&chirpRow)
	if err != nil {
		log.Printf("Error marshaling: %v\n", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("Error marshaling"))
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(dat)

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

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("only devs can call this endpoint\n"))
		return
	}
	cfg.fileserverHits.Store(0)
	err := cfg.db.RemoveUsers(r.Context())
	if err != nil {
		log.Println("could not remove all users")
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("could not remove all users"))
		return
	}

}

func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	type reqBody struct {
		Password      string `json:"password"`
		Email         string `json:"email"`
		ExpireSeconds *int   `json:"expires_in_seconds,omitempty"`
	}

	decoder := json.NewDecoder(r.Body)
	body := reqBody{}
	err := decoder.Decode(&body)

	if err != nil {
		w.WriteHeader(400)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("could not decode request body"))
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		w.WriteHeader(401)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("incorrect email or password"))
		return
	}

	var expireSeconds time.Duration
	if body.ExpireSeconds == nil || *body.ExpireSeconds > 3600 {
		expireSeconds = time.Hour
	} else if *body.ExpireSeconds > 0 {
		expireSeconds = time.Duration(*body.ExpireSeconds)
	} else {
		w.WriteHeader(400)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("invalid expiration time, time must be positive"))
		return
	}
	token, err := auth.MakeJWT(user.ID, cfg.secret, expireSeconds)
	if err != nil {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("could not make JWT"))
		return
	}

	err = auth.CheckPasswordHash(user.HashedPassword, body.Password)
	if err != nil {
		w.WriteHeader(401)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("incorrect email or password"))
		return
	}

	noPass := database.CreateUserRow{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email}

	type ret struct {
		Token string `json:"token"`
		database.CreateUserRow
	}

	retStruct := ret{Token: token, CreateUserRow: noPass}
	dat, err := json.Marshal(&retStruct)
	if err != nil {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("Error marshaling"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)

}

func (cfg *apiConfig) usersHandler(w http.ResponseWriter, r *http.Request) {
	type reqBody struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	body := reqBody{}
	err := decoder.Decode(&body)

	if err != nil {
		w.WriteHeader(400)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("could not decode request body"))
		return
	}

	hpass, err := auth.HashPassword(body.Password)
	if err != nil {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("unable to hash your password"))
		log.Printf("unable to hash your password: %v", err)
		return
	}

	args := database.CreateUserParams{Email: body.Email, HashedPassword: hpass}

	user, err := cfg.db.CreateUser(r.Context(), args)
	if err != nil {
		log.Printf("user email is in use: %v", body.Email)
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("user email is in use"))
		return
	}

	dat, err := json.Marshal(&user)
	if err != nil {
		log.Printf("Error marshaling: %v\n", err)
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("Error marshaling"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(dat)

}
