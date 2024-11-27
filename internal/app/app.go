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

	"github.com/herrmannplatz/go-server/internal/config"
	"github.com/herrmannplatz/go-server/internal/database"
	"github.com/herrmannplatz/go-server/internal/repository"
	"github.com/herrmannplatz/go-server/pkg/util"
)

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
			util.WriteJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}

		util.WriteJSON(w, http.StatusOK, posts)
	})

	a.router.HandleFunc("POST /upload", func(w http.ResponseWriter, r *http.Request) {
		dst, err := os.Create(filepath.Join("./", "example.png"))
		if err != nil {
			util.WriteJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer dst.Close()

		_, err = io.Copy(dst, r.Body)
		if err != nil {
			util.WriteJSONError(w, http.StatusInternalServerError, err.Error())
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
