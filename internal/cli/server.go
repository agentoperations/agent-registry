package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agentregistry/agent-registry/internal/server"
	"github.com/agentregistry/agent-registry/internal/service"
	"github.com/agentregistry/agent-registry/internal/store"
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Server management",
	}
	cmd.AddCommand(newServerStartCmd())
	return cmd
}

func newServerStartCmd() *cobra.Command {
	var port int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the registry server",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := store.NewSQLiteStore(dbPath)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer db.Close()

			svc := service.New(db)
			router := server.NewRouter(svc, uiFS)

			srv := &http.Server{
				Addr:    fmt.Sprintf(":%d", port),
				Handler: router,
			}

			go func() {
				fmt.Printf("Agent Registry server listening on :%d\n", port)
				fmt.Printf("  API: http://localhost:%d/api/v1\n", port)
				fmt.Printf("  UI:  http://localhost:%d\n", port)
				fmt.Printf("  DB:  %s\n", dbPath)
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
					os.Exit(1)
				}
			}()

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit

			fmt.Println("\nShutting down...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return srv.Shutdown(ctx)
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "Server port")
	cmd.Flags().StringVar(&dbPath, "db", "registry.db", "SQLite database path")
	return cmd
}
