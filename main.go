package main

import (
	"context"
	"embed"
	"log"

	"github.com/herrmannplatz/go-server/internal/app"
	"github.com/herrmannplatz/go-server/internal/config"
	_ "github.com/joho/godotenv/autoload"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	app := app.New(cfg, embedMigrations)
	log.Fatal(app.Start(context.Background()))
}
