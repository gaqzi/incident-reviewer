package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/reviewing"
	"github.com/gaqzi/incident-reviewer/internal/reviewing/storage"
	"github.com/gaqzi/incident-reviewer/test/a"
)

func TestMemoryStore(t *testing.T) {
	StorageTest(t, context.Background(), func() reviewing.Storage { return storage.NewMemoryStore() })
}

// StorageTest is a base suite used to test across the implementations of reviewing.Storage.
// It's implemented this way to ensure that the implementations can be used interchangeably, and to allow for the use
// of lighter implementations during testing.
func StorageTest(t *testing.T, ctx context.Context, storeFactory func() reviewing.Storage) {
	t.Run("Save", func(t *testing.T) {
		t.Run("with an empty incident object, it sets an id, created & updated at, and saves it", func(t *testing.T) {
			incident := reviewing.Review{}
			store := storeFactory()

			actual, err := store.Save(ctx, incident)
			require.NoError(t, err, "expected to have saved the object")

			require.Equal(
				t,
				reviewing.Review{
					ID:        actual.ID,
					CreatedAt: actual.CreatedAt,
					UpdatedAt: actual.UpdatedAt,
				},
				actual,
				"expected to have set the id, created at, and updated at fields on save",
			)
			require.NotEmpty(t, actual.ID)
			require.NotEmpty(t, actual.CreatedAt)
			require.NotEmpty(t, actual.UpdatedAt)
		})

		t.Run("when saving an object later it will record a new UpdatedAt", func(t *testing.T) {
			review := a.Review().IsNotSaved().Build()
			store := storeFactory()
			first, err := store.Save(ctx, review)
			require.NoError(t, err)

			// postgres has microsecond resolution, so need to make sure enough time passes.
			time.Sleep(time.Microsecond)
			second, err := store.Save(ctx, first)
			require.NoError(t, err)

			elapsed := second.UpdatedAt.Sub(first.UpdatedAt)
			require.GreaterOrEqual(t, elapsed, time.Nanosecond*10, "expected at least ten nanoseconds to have passed between both")
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("returns an error when an item with the given PK doesn't exist in the store", func(t *testing.T) {
			store := storeFactory()

			_, err := store.Get(ctx, 1_000)
			require.Error(t, err, "expected to not have found an item when it's not in the store")

			var actualErr *storage.NoReviewError
			require.ErrorAs(t, err, &actualErr, "expected the specific error for not found")
		})

		t.Run("after saving, gets back the same object as save when asking by ID", func(t *testing.T) {
			store := storeFactory()
			expected, err := store.Save(ctx, a.Review().IsNotSaved().Build())
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
			review, err := store.Save(ctx, a.Review().IsNotSaved().Build())
			require.NoError(t, err, "expected to have saved successfully")

			actual, err := store.All(ctx)
			require.NoError(t, err)

			require.NotEmpty(t, actual)
			require.Equal(
				t,
				[]reviewing.Review{review},
				actual,
				"expected to have gotten back an item matching the only stored one",
			)
		})

		t.Run("with multiple reviews, returns them in descending creation order", func(t *testing.T) {
			store := storeFactory()
			review1, err := store.Save(ctx, a.Review().IsNotSaved().Build())
			require.NoError(t, err, "expected to have saved successfully")
			review2, err := store.Save(ctx, a.Review().IsNotSaved().Modify(func(r *reviewing.Review) {
				r.URL = "https://example.com/reviews/2"
				r.Title = "Another review"
			}).Build())
			require.NoError(t, err)

			actual, err := store.All(ctx)
			require.NoError(t, err)

			require.Equal(
				t,
				[]reviewing.Review{
					review2,
					review1,
				},
				actual,
				"expected the most recently created item to be returned first",
			)
		})
	})
}
