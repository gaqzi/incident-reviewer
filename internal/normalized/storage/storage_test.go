package storage_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
	"github.com/gaqzi/incident-reviewer/internal/normalized/storage"
)

func TestCauseMemoryStore(t *testing.T) {
	ContributingCauseStorageTest(t, context.Background(), func() normalized.ContributingCauseStorage {
		return storage.NewContributingCauseMemoryStore()
	})
}

func ContributingCauseStorageTest(t *testing.T, ctx context.Context, storeFactory func() normalized.ContributingCauseStorage) {
	validCause := func(modifications ...func(cc *normalized.ContributingCause)) normalized.ContributingCause {
		cc := normalized.ContributingCause{
			Name:        "Third Party Outage",
			Description: "When things go wrong for others",
			Category:    "Design", // because we can mitigate these by designing differently, mostly
		}

		for _, m := range modifications {
			m(&cc)
		}

		return cc
	}

	t.Run("Save", func(t *testing.T) {
		t.Run("an object with all fields set correctly, it saves without an error and a PK is set", func(t *testing.T) {
			cause := validCause()
			store := storeFactory()

			actual, err := store.Save(ctx, cause)

			require.NoError(t, err, "expected to have saved when all fields are set")
			require.NotEmpty(t, actual.ID, "expected the ID to be set to something when saved, is there an error that wasn't covered?")
			require.NotEmpty(t, actual.CreatedAt, "expected to have set CreatedAt when creating since it was empty")
			require.NotEmpty(t, actual.UpdatedAt, "expected to have set the updated at when saving")
			require.Equal(
				t,
				normalized.ContributingCause{
					ID:          actual.ID,
					Name:        cause.Name,
					Description: cause.Description,
					Category:    cause.Category,
					CreatedAt:   actual.CreatedAt,
					UpdatedAt:   actual.UpdatedAt,
				},
				actual,
				"expected to have saved the values as passed in, and set the generated or automatic values",
			)
		})

		t.Run("with an empty cause it fails becasue the required fields are empty", func(t *testing.T) {
			cause := normalized.ContributingCause{}
			store := storeFactory()

			_, err := store.Save(ctx, cause)
			require.Error(t, err, "expected an error from failing validation")

			var errs validator.ValidationErrors
			require.True(t, errors.As(err, &errs), "expected to have converted to validator.ValidationErrors")
			require.GreaterOrEqual(t, len(errs), 3, "expected to have at least 2 fields failing at hte time of writing: %s", err)
		})

		t.Run("when saving an item later it will record a new UpdatedAt", func(t *testing.T) {
			cause := validCause()
			store := storeFactory()
			first, err := store.Save(ctx, cause)
			require.NoError(t, err)

			// postgres has microsecond resolution, so need to make sure enough time passes.
			time.Sleep(time.Microsecond)
			second, err := store.Save(ctx, first)
			require.NoError(t, err)

			elapsed := second.UpdatedAt.Sub(first.UpdatedAt)
			require.GreaterOrEqual(t, elapsed, time.Nanosecond*10, "expected at least ten nanoseconds to have passed between both")

		})

		t.Run("Get", func(t *testing.T) {
			t.Run("returns an error when an item with the given PK doesn't exist in the store", func(t *testing.T) {
				store := storeFactory()

				_, err := store.Get(ctx, 1_000)
				require.Error(t, err, "expected to not have found an item when it's not in the store")

				var actualErr *storage.NoContributingCauseError
				require.ErrorAs(t, err, &actualErr, "expected the specific error for not found")
			})

			t.Run("after saving, gets back the same object as save when asking by ID", func(t *testing.T) {
				store := storeFactory()
				expected, err := store.Save(ctx, validCause())
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
				cause, err := store.Save(ctx, validCause())
				require.NoError(t, err, "expected to have saved successfully")

				actual, err := store.All(ctx)
				require.NoError(t, err)

				require.NotEmpty(t, actual)
				require.Equal(
					t,
					[]normalized.ContributingCause{cause},
					actual,
					"expected to have gotten back an item matching the only stored one",
				)
			})

			t.Run("with multiple reviews, returns them in descending creation order", func(t *testing.T) {
				store := storeFactory()
				cause1, err := store.Save(ctx, validCause())
				require.NoError(t, err, "expected to have saved successfully")
				cause2, err := store.Save(ctx, validCause(func(r *normalized.ContributingCause) {
					r.Name = "Government instability"
					r.Category = "Unavoidable"
				}))
				require.NoError(t, err)

				actual, err := store.All(ctx)
				require.NoError(t, err)

				require.Equal(
					t,
					[]normalized.ContributingCause{
						cause2,
						cause1,
					},
					actual,
					"expected the most recently created item to be returned first",
				)
			})
		})
	})
}
