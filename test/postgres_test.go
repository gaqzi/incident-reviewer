package test_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-sqlx/sqlx"
	_ "github.com/lib/pq"

	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/test"
)

func TestStartPostgres(t *testing.T) {
	t.Run("starts postgres, and returns a valid connection string, and a done to stop postgres", func(t *testing.T) {
		ctx := context.Background()
		psqlCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		err, conn, done := test.StartPostgres(psqlCtx)
		require.NoError(t, err, "expected to have started a postgres container successfully")

		db, err := sqlx.Connect("postgres", conn)
		require.NoError(t, err)

		row := db.QueryRow(`SELECT 1+1`)
		var result int
		require.NoError(t, row.Scan(&result))
		// is it valuable to check for this? it feels like a good safety net but valuable…?
		require.Equal(t, 2, result, "expected to have gotten a valid result from postgres")

		// Now, let's clean up by shutting down the container, any queries after that have to fail.
		done()
		err = db.QueryRow(`SELECT 1+2`).Err()
		require.Error(t, err, "expected an error because calling `done()` shuts down the container")
		require.ErrorContains(t, err, "connect: connection refused", "failed for other reason than container going down")
	})
}
