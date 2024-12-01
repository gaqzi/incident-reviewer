package storage_test

import (
	"context"
	"testing"

	"github.com/gaqzi/incident-reviewer/internal/reviewing"
	"github.com/gaqzi/incident-reviewer/internal/reviewing/storage"
)

func TestMemoryStore(t *testing.T) {
	StorageTest(t, context.Background(), func() reviewing.Storage { return storage.NewMemoryStore() })
}
