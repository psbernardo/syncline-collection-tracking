package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/config"
	"github.com/psbernardo/syncline-collection-tracking/internal/platform/database"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/accounts"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/dashboard"
	"github.com/psbernardo/syncline-collection-tracking/internal/slices/receivables"
	webstatic "github.com/psbernardo/syncline-collection-tracking/internal/web/static"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, closeDB, err := database.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeDB()

	dashboardRepository := dashboard.NewGormRepository(db)
	accountHandler, err := accounts.NewHandler(accounts.NewService(db, accounts.NewGormRepository(db)), dashboard.NewCompanyTotalsService(dashboardRepository))
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	accountHandler.RegisterRoutes(mux)
	accountRepository := accounts.NewGormRepository(db)
	receivableHandler, err := receivables.NewHandler(receivables.NewService(db, receivables.NewGormRepository(db), accountRepository))
	if err != nil {
		return err
	}
	receivableHandler.RegisterRoutes(mux)
	dashboardHandler, err := dashboard.NewHandler(dashboard.NewService(dashboardRepository))
	if err != nil {
		return err
	}
	dashboardHandler.RegisterRoutes(mux)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(webstatic.FS))))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/accounts", http.StatusSeeOther)
	})

	server := &http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           requestTimeouts(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Printf("collection tracking listening on http://%s\n", server.Addr)
	return server.ListenAndServe()
}

func requestTimeouts(next http.Handler) http.Handler {
	return http.TimeoutHandler(next, 15*time.Second, "request timed out")
}
