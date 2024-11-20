package test

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/go-sqlx/sqlx"
	"github.com/pressly/goose/v3"
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

	if err := migrate(connectionString); err != nil {
		return err, "", func() {}
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

type gooseErrorLogger struct{}

func (*gooseErrorLogger) Fatalf(format string, v ...interface{}) { log.Fatalf(format, v...) }
func (*gooseErrorLogger) Printf(format string, v ...interface{}) { /* noop */ }

// migrate the database using goose.
// Because we're using goose like this we pull in all of its dependencies
// indirectly, which are a few of them, but since it's only used in the test
// package we should be fine and not leak out.
func migrate(conn string) error {
	db, err := sqlx.Connect("postgres", conn)
	if err != nil {
		return fmt.Errorf("failed to connect to the DB: %w", err)
	}
	defer (func() { _ = db.Close() })()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to configure goose for postgres: %w", err)
	}

	// Don't output stuff when it's working as expected
	goose.SetLogger(&gooseErrorLogger{})

	if err := goose.Up(db.DB, "../migrations"); err != nil {
		return fmt.Errorf("failed to migrate using goose: %w", err)
	}

	return nil
}
