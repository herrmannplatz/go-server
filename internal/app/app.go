package app

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"

	"github.com/herrmannplatz/chirpy/internal/config"
	"github.com/herrmannplatz/chirpy/internal/database"
	"github.com/herrmannplatz/chirpy/internal/repository"
	"github.com/herrmannplatz/chirpy/pkg/util"
	"golang.org/x/tools/go/cfg"
)

type responseError struct {
	Error string `json:"error"`
}

type App struct {
	router     *http.ServeMux
	migrations embed.FS
}

func New(cfg config.Config, migrations embed.FS) *App {
	return &App{
		router:     http.NewServeMux(),
		migrations: migrations,
	}
}

func (a *App) Start(ctx context.Context) error {
	db, err := database.ConnectToDB(cfg.DatabaseUrl, a.migrations)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}
	defer db.Close()

	queries := repository.New(db)

	a.router.HandleFunc("GET /posts", func(w http.ResponseWriter, r *http.Request) {
		type Post struct {
			ID    int32  `json:"id"`
			Title string `json:"title"`
			Body  string `json:"body"`
		}

		posts, err := queries.GetPosts(ctx)
		if err != nil {
			util.SendJSON(w, http.StatusInternalServerError, responseError{Error: err.Error()})
			return
		}

		response := []Post{}
		for _, post := range posts {
			response = append(response, Post{
				ID:    post.ID,
				Title: post.Title,
				Body:  post.Body,
			})
		}

		util.SendJSON(w, http.StatusOK, response)
	})

	server := &http.Server{
		Addr:    ":" + cfg.PORT,
		Handler: a.router,
	}

	log.Printf("Starting server on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}
