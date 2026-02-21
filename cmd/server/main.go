package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agentregistry/agent-registry/internal/server"
	"github.com/agentregistry/agent-registry/internal/service"
	"github.com/agentregistry/agent-registry/internal/store"
)

//go:embed ui
var uiEmbed embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "registry.db"
	}

	db, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	uiContent, _ := fs.Sub(uiEmbed, "ui")
	svc := service.New(db)
	router := server.NewRouter(svc, uiContent)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		fmt.Printf("Agent Registry server listening on :%s\n", port)
		fmt.Printf("  API: http://localhost:%s/api/v1\n", port)
		fmt.Printf("  UI:  http://localhost:%s\n", port)
		fmt.Printf("  DB:  %s\n", dbPath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
