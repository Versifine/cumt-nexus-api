package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Versifine/cumt-nexus-api/internal/admin/adminrepository"
	"github.com/Versifine/cumt-nexus-api/internal/admin/adminusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "admin command failed: %v\n", err)
		os.Exit(1)
	}
}

func run(command string, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.App.StartupTimeout)
	defer cancel()

	pool, err := db.Open(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer db.Close(pool)
	if err := db.Ping(ctx, pool); err != nil {
		return err
	}

	uc := adminusecase.NewUseCase(adminrepository.NewPostgresAdminRepository(pool), nil)
	switch command {
	case "bootstrap-owner":
		return runBootstrapOwner(ctx, uc, args)
	case "recover-owner":
		return runRecoverOwner(ctx, uc, args)
	default:
		usage()
		return fmt.Errorf("unknown command %q", command)
	}
}

func runBootstrapOwner(ctx context.Context, uc *adminusecase.UseCase, args []string) error {
	fs := flag.NewFlagSet("bootstrap-owner", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	userID := fs.String("user-id", "", "active user id to bootstrap as platform owner")
	reason := fs.String("reason", "", "audit reason")
	confirm := fs.Bool("confirm", false, "confirm the owner bootstrap")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := uc.BootstrapOwner(ctx, adminusecase.BootstrapOwnerInput{
		UserID:  *userID,
		Reason:  *reason,
		Confirm: *confirm,
	})
	if err != nil {
		return err
	}
	fmt.Printf("bootstrapped owner: %s (%s)\n", result.User.Username, result.User.ID)
	return nil
}

func runRecoverOwner(ctx context.Context, uc *adminusecase.UseCase, args []string) error {
	fs := flag.NewFlagSet("recover-owner", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	newOwnerID := fs.String("new-owner-user-id", "", "active user id to recover as platform owner")
	compromisedID := fs.String("compromised-user-id", "", "compromised platform owner user id")
	reason := fs.String("reason", "", "audit reason")
	revokeSessions := fs.Bool("revoke-sessions", false, "revoke compromised owner tokens")
	disableCompromised := fs.Bool("disable-compromised", false, "disable compromised owner account")
	confirm := fs.Bool("confirm", false, "confirm the owner recovery")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := uc.RecoverOwner(ctx, adminusecase.RecoverOwnerInput{
		NewOwnerUserID:     *newOwnerID,
		CompromisedUserID:  *compromisedID,
		Reason:             *reason,
		RevokeSessions:     *revokeSessions,
		DisableCompromised: *disableCompromised,
		Confirm:            *confirm,
	})
	if err != nil {
		return err
	}
	fmt.Printf("recovered owner: %s (%s); compromised user: %s (%s)\n", result.NewOwner.Username, result.NewOwner.ID, result.CompromisedUser.Username, result.CompromisedUser.ID)
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/admin bootstrap-owner --user-id <uuid> --reason <text> --confirm")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/admin recover-owner --new-owner-user-id <uuid> --compromised-user-id <uuid> --reason <text> --revoke-sessions --disable-compromised --confirm")
}
