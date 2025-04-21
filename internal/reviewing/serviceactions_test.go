package reviewing

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
)

func TestActionMapper(t *testing.T) {
	t.Run("the default mapper covers all actions with custom behavior on the Review", func(t *testing.T) {
		mapper := reviewServiceActions()

		require.ElementsMatch(
			t,
			[]string{
				"BindContributingCause",
				"UpdateBoundContributingCause",
				"Save",
				"BindTrigger",
			},
			mapper.All(),
			"expected all causes to be listed here so we catch when we add new or remove one",
		)
	})

	t.Run("BindContributingCause sets the contributing.Cause on the BoundCause before adding it to the Review and sets a valid ID if not provided", func(t *testing.T) {
		mapper := reviewServiceActions()

		doer, err := mapper.Get("BindContributingCause")
		require.NoError(t, err)
		do, ok := doer.(func(Review, contributing.Cause, BoundCause) (Review, error))
		require.True(t, ok)

		cause := contributing.Cause{Name: "Something"}
		review, err := do(Review{}, cause, BoundCause{})
		require.NoError(t, err)

		require.Equal(
			t,
			Review{BoundCauses: []BoundCause{{ID: review.BoundCauses[0].ID, Cause: cause}}},
			review,
			"expected the contributing cause to have been set on the BoundCause and then added to the Review",
		)
	})

	t.Run("BindTrigger sets the normalized.Trigger on the BoundTrigger before adding it to the Review and sets a valid ID if not provided", func(t *testing.T) {
		mapper := reviewServiceActions()

		doer, err := mapper.Get("BindTrigger")
		require.NoError(t, err)
		do, ok := doer.(func(Review, normalized.Trigger, UnboundTrigger) (Review, error))
		require.True(t, ok)

		trigger := normalized.Trigger{Name: "Something"}
		review, err := do(Review{}, trigger, UnboundTrigger{Why: "a good reason"})
		require.NoError(t, err)

		require.Equal(
			t,
			Review{BoundTriggers: []BoundTrigger{{ID: review.BoundTriggers[0].ID, Trigger: trigger, UnboundTrigger: UnboundTrigger{Why: "a good reason"}}}},
			review,
			"expected the contributing trigger to have been set on the BoundTrigger and then added to the Review",
		)
	})

	t.Run("Save validates and returns an error when it fails to validate", func(t *testing.T) {
		mapper := reviewServiceActions()

		doer, actual := mapper.Get("Save")
		require.NoError(t, actual)
		do, ok := doer.(func(context.Context, Review) (Review, error))
		require.True(t, ok)

		_, actual = do(context.Background(), Review{})

		var errs validator.ValidationErrors
		require.ErrorAs(t, actual, &errs, "expected to have gotten back validation errors")
		require.ErrorContains(t, actual, "failed to validate review:")
		require.GreaterOrEqual(t, len(errs), 8, "expected at minimum 8 errors to match the fields at the time of writing")
	})

	t.Run("Save updates the timestamps after successfully validating", func(t *testing.T) {
		mapper := reviewServiceActions()
		doer, actual := mapper.Get("Save")
		require.NoError(t, actual)
		do, ok := doer.(func(context.Context, Review) (Review, error))
		require.True(t, ok)

		validReview := func() Review {
			return Review{
				ID:                  uuid.Must(uuid.NewV7()),
				URL:                 "http://example.com",
				Title:               "example",
				Description:         "example",
				Impact:              "example",
				Where:               "example",
				ReportProximalCause: "example",
				ReportTrigger:       "example",
			}
		}

		t.Run("when timestamps are zero both are set to now", func(t *testing.T) {
			r := validReview()

			r, err := do(context.Background(), r)

			require.NoError(t, err)
			require.NotZero(t, r.CreatedAt, "expected to have been set")
			require.Equal(t, r.CreatedAt, r.UpdatedAt, "expected to have been set to the same when both are blank")
		})

		t.Run("when created at already is set then only updated at is updated", func(t *testing.T) {
			r := validReview()
			now := time.Now()
			r.CreatedAt = now

			time.Sleep(time.Millisecond) // to give us some time to make the comparisons work
			r, err := do(context.Background(), r)

			require.NoError(t, err)
			require.Greater(t, r.UpdatedAt.UnixNano(), r.CreatedAt.UnixNano(), "expected the updated at to be after created at")
		})
	})
}
