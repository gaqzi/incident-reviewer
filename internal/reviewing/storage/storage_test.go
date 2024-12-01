package storage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/reviewing"
	"github.com/gaqzi/incident-reviewer/internal/reviewing/storage"
)

// StorageTest is a base suite used to test across the implementations of reviewing.Storage.
// It's implemented this way to ensure that the implementations can be used interchangeably, and to allow for the use
// of lighter implementations during testing.
func StorageTest(t *testing.T, ctx context.Context, storeFactory func() reviewing.Storage) {
	validReview := func() reviewing.Review {
		return reviewing.Review{
			URL:         "https://example.com/reviews/1",
			Title:       "Something",
			Description: "At the bottom of the sea",
			Impact:      "did a bunch of things",
		}
	}

	t.Run("Save", func(t *testing.T) {
		t.Run("an object with all fields set correctly, it saves without an error and a PK is set", func(t *testing.T) {
			review := validReview()
			store := storeFactory()
			actual, err := store.Save(ctx, review)

			require.NoError(t, err, "expected to have saved successfully when all fiels are set correctly")
			require.NotEmpty(t, actual.ID, "expected the ID to be set to something when saved, is there an error that wasn't covered?")
			require.Equal(
				t,
				reviewing.Review{
					ID:          actual.ID,
					URL:         review.URL,
					Title:       review.Title,
					Description: review.Description,
					Impact:      review.Impact,
				},
				actual,
				"expected to have saved and set the ID which is used as primary key",
			)
		})

		t.Run("with an empty incident object, it fails because the required fields are empty", func(t *testing.T) {
			incident := reviewing.Review{}
			store := storeFactory()

			_, err := store.Save(ctx, incident)
			require.Error(t, err, "expected errors because the validation isn't valid")

			var errs validator.ValidationErrors
			require.True(t, errors.As(err, &errs), "expected to have converted to validator.ValidationErrors")
			require.GreaterOrEqualf(t, len(errs), 4, "expected at least the 4 fields at the time of writing as failing: %s", errs)
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
			expected, err := store.Save(ctx, validReview())
			require.NoError(t, err, "expected the valid review to have been saved successfully")

			actual, err := store.Get(ctx, expected.ID)
			require.NoError(t, err, "expected to have fetched successfully when just saving the object")

			require.Equal(t, actual, expected, "expected the objects to have the same info when no changes between save and fetch")
		})
	})
}
