package main

import (
	"database/sql"
	"embed"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/herrmannplatz/chirpy/internal/api"
	"github.com/herrmannplatz/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

//go:embed sql/schema/*.sql
var embedMigrations embed.FS

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func main() {
	godotenv.Load()

	goose.SetBaseFS(embedMigrations)

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	if err := goose.Up(db, "sql/schema"); err != nil {
		log.Fatal(err)
	}

	apiCfg := api.Config{
		FileserverHits: atomic.Int32{},
		Db:             database.New(db),
		Platform:       os.Getenv("PLATFORM"),
		Secret:         os.Getenv("JWT_SECRET"),
		POLKA_KEY:      os.Getenv("POLKA_KEY"),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", handlerReadiness)

	mux.HandleFunc("POST /api/chirps", apiCfg.HandlerPostChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.HandlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.HandlerGetChirp)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.HandlerDeleteChirp)

	mux.HandleFunc("PUT /api/users", apiCfg.HandlerPutUsers)
	mux.HandleFunc("POST /api/users", apiCfg.HandlerPostUsers)

	mux.HandleFunc("POST /api/login", apiCfg.HandlerLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.HandlerRefreshToken)
	mux.HandleFunc("POST /api/revoke", apiCfg.HandlerRevokeToken)

	mux.HandleFunc("POST /admin/reset", apiCfg.HandlerReset)

	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.HandlerPolkaWebhook)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	log.Printf("Starting server on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
