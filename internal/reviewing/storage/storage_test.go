package storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/reviewing"
	"github.com/gaqzi/incident-reviewer/internal/reviewing/storage"
)

// StorageTest is a base suite used to test across the implementations of reviewing.Storage.
// It's implemented this way to ensure that the implementations can be used interchangeably, and to allow for the use
// of lighter implementations during testing.
func StorageTest(t *testing.T, ctx context.Context, storeFactory func() reviewing.Storage) {
	validReview := func(modify ...func(r *reviewing.Review)) reviewing.Review {
		review := reviewing.Review{
			URL:                 "https://example.com/reviews/1",
			Title:               "Something",
			Description:         "At the bottom of the sea",
			Impact:              "did a bunch of things",
			Where:               "At land",
			ReportProximalCause: "Broken",
			ReportTrigger:       "Special operation",
		}

		for _, mod := range modify {
			mod(&review)
		}

		return review
	}

	t.Run("Save", func(t *testing.T) {
		t.Run("an object with all fields set correctly, it saves without an error and a PK is set", func(t *testing.T) {
			review := validReview()
			store := storeFactory()
			actual, err := store.Save(ctx, review)

			require.NoError(t, err, "expected to have saved successfully when all fiels are set correctly")
			require.NotEmpty(t, actual.ID, "expected the ID to be set to something when saved, is there an error that wasn't covered?")
			require.NotEmpty(t, actual.CreatedAt, "expected to have set CreatedAt when creating if it was empty")
			require.NotEmpty(t, actual.UpdatedAt, "expected to have set UpdatedAt when saving")
			require.Equal(
				t,
				reviewing.Review{
					ID:                  actual.ID,
					URL:                 review.URL,
					Title:               review.Title,
					Description:         review.Description,
					Impact:              review.Impact,
					Where:               review.Where,
					ReportProximalCause: review.ReportProximalCause,
					ReportTrigger:       review.ReportTrigger,
					CreatedAt:           actual.CreatedAt,
					UpdatedAt:           actual.UpdatedAt,
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

		t.Run("when saving an object later it will record a new UpdatedAt", func(t *testing.T) {
			review := validReview()
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
			expected, err := store.Save(ctx, validReview())
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
			review, err := store.Save(ctx, validReview())
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
			review1, err := store.Save(ctx, validReview())
			require.NoError(t, err, "expected to have saved successfully")
			review2, err := store.Save(ctx, validReview(func(r *reviewing.Review) {
				r.URL = "https://example.com/reviews/2"
				r.Title = "Another review"
			}))
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
