package test

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func StartPostgres(ctx context.Context) (err error, conn string, done func()) {
	migrationFiles, err := filepath.Glob("../migrations/*.sql")
	if err != nil {
		return fmt.Errorf("failed to list database migrations: %w", err), "", func() {}
	}

	postgresContainer, err := postgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase("incident_reviewer"),
		postgres.WithInitScripts(migrationFiles...),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to start postgres: %w", err), "", nil
	}

	connectionString, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get postgres connection string: %w", err), "", nil
	}

	return nil, connectionString, func() {
		// the below line is for the next version of testcontainers, but it was already part of the docs,
		// so should move there when I upgrade.
		// if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
		if err := postgresContainer.Terminate(context.Background()); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}
}
