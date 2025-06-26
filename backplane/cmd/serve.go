package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/heldtogether/traintrack/internal/router"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the traintrack backplane server",
	Run: func(cmd *cobra.Command, args []string) {
		RunServe()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func RunServe() error {
	// Load .env file only if TRAINTRACK_DATABASE_URL is not already set
	if os.Getenv("TRAINTRACK_DATABASE_URL") == "" {
		err := godotenv.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading .env file: %v\n", err)
			os.Exit(1)
		}
	}

	conn, err := pgxpool.New(context.Background(), os.Getenv("TRAINTRACK_DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get migration location: %s", err)
	}
	path := "file://" + filepath.Join(cwd, "migrations")

	m, err := migrate.New(
		path,
		os.Getenv("TRAINTRACK_DATABASE_URL"),
	)
	if err != nil {
		log.Fatalf("could not migrate db: %s", err)
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration failed: %s", err)
	} else if err == migrate.ErrNoChange {
		log.Println("no new migrations to apply")
	} else {
		log.Println("migrations applied successfully")
	}

	router := router.Setup(conn)
	return http.ListenAndServe(":8080", router)
}
