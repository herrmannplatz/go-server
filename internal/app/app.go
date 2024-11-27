package app

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/herrmannplatz/chirpy/internal/config"
	"github.com/herrmannplatz/chirpy/internal/database"
	"github.com/herrmannplatz/chirpy/internal/repository"
	"github.com/herrmannplatz/chirpy/pkg/util"
)

type responseError struct {
	Error string `json:"error"`
}

type App struct {
	cfg        config.Config
	router     *http.ServeMux
	migrations embed.FS
}

func New(cfg config.Config, migrations embed.FS) *App {
	return &App{
		cfg:        cfg,
		router:     http.NewServeMux(),
		migrations: migrations,
	}
}

func (a *App) Start(ctx context.Context) error {
	db, err := database.ConnectToDB(a.cfg.DatabaseUrl, a.migrations)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}
	defer db.Close()

	queries := repository.New(db)

	a.router.HandleFunc("GET /posts", func(w http.ResponseWriter, r *http.Request) {
		posts, err := queries.GetPosts(ctx)
		if err != nil {
			util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
			return
		}

		util.SendJSON(w, http.StatusOK, posts)
	})

	a.router.HandleFunc("POST /upload", func(w http.ResponseWriter, r *http.Request) {
		dst, err := os.Create(filepath.Join("./", "example.png"))
		if err != nil {
			util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
			return
		}
		defer dst.Close()

		_, err = io.Copy(dst, r.Body)
		if err != nil {
			util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
			return
		}

		fmt.Println("File uploaded successfully")
	})

	server := &http.Server{
		Addr:    ":" + a.cfg.PORT,
		Handler: a.router,
	}

	log.Printf("Starting server on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}
