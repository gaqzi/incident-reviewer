package storage_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	storage2 "github.com/gaqzi/incident-reviewer/internal/normalized/contributing/storage"
	"github.com/gaqzi/incident-reviewer/internal/normalized/storage"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
	"github.com/gaqzi/incident-reviewer/test/a"
)

func TestTriggerMemoryStore(t *testing.T) {
	TriggerStorageTest(t, context.Background(), func() normalized.TriggerStorage {
		return storage.NewTriggerMemoryStore()
	})
}

func TriggerStorageTest(t *testing.T, ctx context.Context, storeFactory func() normalized.TriggerStorage) {
	t.Run("Save", func(t *testing.T) {
		t.Run("returns an error when trying to save without an ID set", func(t *testing.T) {
			store := storeFactory()

			_, actual := store.Save(ctx, normalized.Trigger{})

			require.ErrorIs(t, actual, storage2.NoIDError, "expected the sentinel error for not having an ID set")
		})

		t.Run("an object with the ID set is saved without errors", func(t *testing.T) {
			Trigger := normalized.NewTrigger()
			store := storeFactory()

			_, err := store.Save(ctx, Trigger)

			require.NoError(t, err, "expected to have saved when all fields are set")
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("returns an error when an item with the given PK doesn't exist in the store", func(t *testing.T) {
			store := storeFactory()

			_, err := store.Get(ctx, uuid.Nil)
			require.Error(t, err, "expected to not have found an item when it's not in the store")

			var actualErr *storage2.NoTriggerError
			require.ErrorAs(t, err, &actualErr, "expected the specific error for not found")
		})

		t.Run("after saving, gets back the same object as save when asking by ID", func(t *testing.T) {
			store := storeFactory()
			expected, err := store.Save(ctx, a.NormalizedTrigger().Build())
			require.NoError(t, err, "expected the valid review to have been saved successfully")

			actual, err := store.Get(ctx, expected.ID)
			require.NoError(t, err, "expected to have fetched successfully when just saving the object")

			require.Equal(t, actual, expected, "expected the objects to have the same info when no changes between save and fetch")
		})
	})

	t.Run("All", func(t *testing.T) {
		t.Run("with no stored reviews it returns an empty list", func(t *testing.T) {
			store := storeFactory()

			reviews, err := store.All(ctx)
			require.NoError(t, err)

			require.Empty(t, reviews, "expected to have gotten back no items")
		})

		t.Run("returns the only stored item when only one exists", func(t *testing.T) {
			store := storeFactory()
			trigger, err := store.Save(ctx, a.NormalizedTrigger().Build())
			require.NoError(t, err, "expected to have saved successfully")

			actual, err := store.All(ctx)
			require.NoError(t, err)

			require.NotEmpty(t, actual)
			require.Equal(
				t,
				[]normalized.Trigger{trigger},
				actual,
				"expected to have gotten back an item matching the only stored one",
			)
		})

		t.Run("with multiple reviews, returns them in descending creation order", func(t *testing.T) {
			store := storeFactory()
			Trigger1, err := store.Save(ctx, a.NormalizedTrigger().Build())
			require.NoError(t, err, "expected to have saved successfully")
			Trigger2, err := store.Save(ctx, a.NormalizedTrigger().WithID(uuid.Must(uuid.NewV7())).WithName("Unbounded resource utilization").Build())
			require.NoError(t, err)

			actual, err := store.All(ctx)
			require.NoError(t, err)

			require.Equal(
				t,
				[]normalized.Trigger{
					Trigger2,
					Trigger1,
				},
				actual,
				"expected the most recently created item to be returned first",
			)
		})
	})
}
