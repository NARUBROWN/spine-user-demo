package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"spine-user-demo/internal/application"
	"spine-user-demo/internal/database"
	"spine-user-demo/migrations"
)

// @title Spine User Demo API
// @version 0.2.0
// @description Spine + Swaggo
// @host localhost:8080
// @BasePath /
func main() {
	migrateOnly := flag.Bool("migrate", false, "Run database migrations and exit")
	flag.Parse()
	if *migrateOnly {
		db, err := database.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Database configuration failed: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()
		if err := migrations.Run(context.Background(), db); err != nil {
			fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Migrations applied successfully.")
		return
	}

	app, options, err := application.BuildApplication()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Application configuration failed: %v\n", err)
		os.Exit(1)
	}
	if err := errors.Join(app.Validate(options), database.ValidateEnvironment()); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(options); err != nil {
		fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}
