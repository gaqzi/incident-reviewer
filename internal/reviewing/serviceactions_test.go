package reviewing

import (
	"context"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
)

func TestActionMapper(t *testing.T) {
	t.Run("the default mapper covers all actions with custom behavior on the Review", func(t *testing.T) {
		mapper := reviewServiceActions()

		require.ElementsMatch(
			t,
			[]string{
				"AddContributingCause",
				"Save",
			},
			mapper.All(),
			"expected all causes to be listed here so we catch when we add new or remove one",
		)
	})

	t.Run("AddContributingCause sets the contributing.Cause on the ReviewCause before adding it to the Review", func(t *testing.T) {
		mapper := reviewServiceActions()

		doer, err := mapper.Get("AddContributingCause")
		require.NoError(t, err)
		do, ok := doer.(func(Review, contributing.Cause, ReviewCause) (Review, error))
		require.True(t, ok)

		cause := contributing.Cause{Name: "Something"}
		review, err := do(Review{}, cause, ReviewCause{})
		require.NoError(t, err)

		require.Equal(
			t,
			Review{ContributingCauses: []ReviewCause{{Cause: cause}}},
			review,
			"expected the contributing cause to have been set on the ReviewCause and then added to the Review",
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

	})
}

// Why is this test here? Because it's testing something _inside_ the reviewing package and
// this is the one test I have that operates in here. It's also why I'm not using the `a` helpers
// because I would be causing a recursive import loop.
func TestReview_updateTimestamps(t *testing.T) {
	validReview := func() Review {
		return Review{
			ID:                  uuid.Nil,
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

		r = r.updateTimestamps()

		require.NotZero(t, r.CreatedAt, "expected to have been set")
		require.Equal(t, r.CreatedAt, r.UpdatedAt, "expected to have been set to the same when both are blank")
	})

	t.Run("when created at already is set then only updated at is updated", func(t *testing.T) {
		r := validReview()
		now := time.Now()
		r.CreatedAt = now

		time.Sleep(time.Millisecond) // to give us some time to make the comparisons work
		r = r.updateTimestamps()

		require.Greater(t, r.UpdatedAt.UnixNano(), r.CreatedAt.UnixNano(), "expected the updated at to be after created at")
	})
}
