package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/psbernardo/syncline-collection-tracking/internal/config"
	"github.com/psbernardo/syncline-collection-tracking/internal/migrations"
)

func main() {
	godotenv.Load()
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg := config.Read()
	tx := cfg.ConnectDB()
	return migrations.Up(ctx, tx)
}
