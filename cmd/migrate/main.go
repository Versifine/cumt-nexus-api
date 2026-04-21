package main

import (
	"fmt"
	"os"

	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	dsn := db.BuildDSN(cfg.Postgres)

	m, err := migrate.New(
		"file://migrations",
		dsn,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating migrate instance: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "version":
		version, dirty, err := m.Version()
		if err == migrate.ErrNilVersion {
			fmt.Println("No migrations applied")
			os.Exit(1)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Version: %d, Dirty: %t\n", version, dirty)
	default:
		usage()
		os.Exit(1)
	}

}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: go run ./cmd/migrate [up|down|version]")
}
