package reviewing_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
	normStore "github.com/gaqzi/incident-reviewer/internal/normalized/storage"
	"github.com/gaqzi/incident-reviewer/internal/reviewing"
	"github.com/gaqzi/incident-reviewer/internal/reviewing/storage"
)

func TestService_AddContributingCause(t *testing.T) {
	t.Run("when review doesn't exist it returns the error from the storage", func(t *testing.T) {
		store := storage.NewMemoryStore()
		causeStore := normStore.NewContributingCauseMemoryStore()
		service := reviewing.NewService(store, causeStore)
		ctx := context.Background()

		actual := service.AddContributingCause(ctx, 1, 1, "because")

		require.Error(t, actual, "expected an error since we haven't stored any reviews")
		require.ErrorContainsf(t, actual, "failed to get review:", "so we know we got the correct error")
	})

	t.Run("when the contributing cause isn't known return the error from it", func(t *testing.T) {
		store := storage.NewMemoryStore()
		causeStore := normStore.NewContributingCauseMemoryStore()
		service := reviewing.NewService(store, causeStore)
		ctx := context.Background()
		// TODO: need to make some generic heplers for creating these objects, probably under the test package?
		review, err := store.Save(ctx, reviewing.Review{
			URL:                 "http://example.com",
			Title:               "Example",
			Description:         "Example",
			Impact:              "Example",
			Where:               "Example",
			ReportProximalCause: "Example",
			ReportTrigger:       "Example",
		})
		require.NoError(t, err, "expected to have saved the review successfully")

		actual := service.AddContributingCause(ctx, review.ID, 1, "because!")

		require.Error(t, actual, "expected an error when invalid cause provided")
		require.ErrorContains(t, actual, "failed to get contributing cause:")
	})

	t.Run("when both review and contributing cause are known bind it", func(t *testing.T) {
		store := storage.NewMemoryStore()
		causeStore := normStore.NewContributingCauseMemoryStore()
		service := reviewing.NewService(store, causeStore)
		ctx := context.Background()
		review, err := store.Save(ctx, reviewing.Review{
			URL:                 "http://example.com",
			Title:               "Example",
			Description:         "Example",
			Impact:              "Example",
			Where:               "Example",
			ReportProximalCause: "Example",
			ReportTrigger:       "Example",
		})
		require.NoError(t, err, "expected to have saved the review successfully")
		cause, err := causeStore.Save(ctx, normalized.ContributingCause{
			Name:        "Example",
			Description: "Example",
			Category:    "Example",
		})
		require.NoError(t, err, "expected the cause to be saved successfully")

		actual := service.AddContributingCause(ctx, review.ID, cause.ID, "because")
		require.NoError(t, actual, "expected to have bound the cause to the review successfully")

		review, err = store.Get(ctx, review.ID)
		require.NoError(t, err)

		require.Equal(
			t,
			[]reviewing.ReviewCause{
				{
					Cause: cause,
					Why:   "because",
				},
			},
			review.ContributingCauses,
			"expected the cause to have been returned with the review",
		)
	})
}
