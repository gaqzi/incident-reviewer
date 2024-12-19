package normalized_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
	"github.com/gaqzi/incident-reviewer/test/a"
)

type causeStorageMock struct {
	mock.Mock
}

func (m *causeStorageMock) Get(ctx context.Context, id int64) (normalized.ContributingCause, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(normalized.ContributingCause), args.Error(1)
}

func (m *causeStorageMock) Save(ctx context.Context, cause normalized.ContributingCause) (normalized.ContributingCause, error) {
	args := m.Called(ctx, cause)
	return args.Get(0).(normalized.ContributingCause), args.Error(1)
}

func (m *causeStorageMock) All(ctx context.Context) ([]normalized.ContributingCause, error) {
	args := m.Called(ctx)
	return args.Get(0).([]normalized.ContributingCause), args.Error(1)
}

func TestContributingCauseService_Save(t *testing.T) {
	t.Run("sets the Created and Updated at when they're not set", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.
			On("Save", mock.Anything, mock.MatchedBy(func(c normalized.ContributingCause) bool {
				return !c.CreatedAt.IsZero() && !c.UpdatedAt.IsZero() && c.CreatedAt.Equal(c.UpdatedAt)
			})).
			Return(normalized.ContributingCause{}, nil)
		service := normalized.NewContributingCauseService(storage)

		_, err := service.Save(context.Background(), a.ContributingCause().IsNotSaved().Build())

		require.NoError(t, err)
	})

	t.Run("sets the UpdatedAt when updating a previously saved item", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.
			On("Save", mock.Anything, mock.MatchedBy(func(c normalized.ContributingCause) bool {
				return !c.UpdatedAt.IsZero() && c.UpdatedAt.After(c.CreatedAt)
			})).
			Return(normalized.ContributingCause{}, nil)
		service := normalized.NewContributingCauseService(storage)

		_, err := service.Save(context.Background(), a.ContributingCause().IsSaved().Build())

		require.NoError(t, err)
	})

	t.Run("validate the ContributingCause object before saving", func(t *testing.T) {
		service := normalized.NewContributingCauseService(nil)

		_, actual := service.Save(context.Background(), normalized.ContributingCause{})

		require.Error(t, actual)
		require.ErrorContains(t, actual, "failed to validate contributing cause:")
		var errs validator.ValidationErrors
		require.ErrorAs(t, actual, &errs)
		require.GreaterOrEqual(t, len(errs), 3, "expected to have at minimum 3 errors for the required fields")
	})

	t.Run("wraps any error from the store and returns it", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.On("Save", mock.Anything, mock.Anything).
			Return(normalized.ContributingCause{}, errors.New("uh-oh"))
		service := normalized.NewContributingCauseService(storage)

		_, err := service.Save(context.Background(), a.ContributingCause().Build())

		require.Error(t, err, "expected to have failed when the underlying storage always fails")
		require.ErrorContains(t, err, "failed to store contributing cause:")
	})

	t.Run("on successful save returns the updated cause", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.On("Save", mock.Anything, mock.Anything).
			Return(a.ContributingCause().Build(), nil)
		service := normalized.NewContributingCauseService(storage)

		actual, err := service.Save(context.Background(), a.ContributingCause().IsNotSaved().Build())

		require.NoError(t, err)
		require.Equal(t, a.ContributingCause().Build(), actual)
	})
}

func TestContributingCauseService_All(t *testing.T) {
	t.Run("returns a wrapped error if one is returned from the storage", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.On("All", mock.Anything).Return(([]normalized.ContributingCause)(nil), errors.New("uh-oh"))
		ctx := context.Background()
		service := normalized.NewContributingCauseService(storage)

		_, actual := service.All(ctx)

		require.Error(t, actual, "expected an error to be returned since it's hardcoded to always give one")
		require.ErrorContains(t, actual, "unable to get all contributing causes from storage:")
	})

	t.Run("returns the returned object when no errors", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.On("All", mock.Anything).Return([]normalized.ContributingCause{{ID: 1}}, nil)
		ctx := context.Background()
		service := normalized.NewContributingCauseService(storage)

		actual, err := service.All(ctx)
		require.NoError(t, err)

		require.Equal(
			t,
			[]normalized.ContributingCause{{ID: 1}},
			actual,
		)
	})
}
