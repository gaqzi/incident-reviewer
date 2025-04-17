package known_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/known"
	"github.com/gaqzi/incident-reviewer/test/a"
)

type causeStorageMock struct {
	mock.Mock
}

func (m *causeStorageMock) Get(ctx context.Context, id uuid.UUID) (known.Cause, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(known.Cause), args.Error(1)
}

func (m *causeStorageMock) Save(ctx context.Context, cause known.Cause) (known.Cause, error) {
	args := m.Called(ctx, cause)
	return args.Get(0).(known.Cause), args.Error(1)
}

func (m *causeStorageMock) All(ctx context.Context) ([]known.Cause, error) {
	args := m.Called(ctx)
	return args.Get(0).([]known.Cause), args.Error(1)
}

func TestKnownCauseService_Save(t *testing.T) {
	t.Run("sets the Created and Updated at when they're not set", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.
			On("Save", mock.Anything, mock.MatchedBy(func(c known.Cause) bool {
				return !c.CreatedAt.IsZero() && !c.UpdatedAt.IsZero() && c.CreatedAt.Equal(c.UpdatedAt)
			})).
			Return(known.Cause{}, nil)
		service := known.NewCauseService(storage)

		_, err := service.Save(context.Background(), a.Cause().IsNotSaved().Build())

		require.NoError(t, err)
	})

	t.Run("sets the UpdatedAt when updating a previously saved item", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.
			On("Save", mock.Anything, mock.MatchedBy(func(c known.Cause) bool {
				return !c.UpdatedAt.IsZero() && c.UpdatedAt.After(c.CreatedAt)
			})).
			Return(known.Cause{}, nil)
		service := known.NewCauseService(storage)

		_, err := service.Save(context.Background(), a.Cause().IsSaved().Build())

		require.NoError(t, err)
	})

	t.Run("validate the Cause object before saving", func(t *testing.T) {
		service := known.NewCauseService(nil)

		_, actual := service.Save(context.Background(), known.Cause{})

		require.Error(t, actual)
		require.ErrorContains(t, actual, "failed to validate known cause:")
		var errs validator.ValidationErrors
		require.ErrorAs(t, actual, &errs)
		require.GreaterOrEqual(t, len(errs), 4, "expected to have at minimum 4 errors for the required fields")
	})

	t.Run("wraps any error from the store and returns it", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.On("Save", mock.Anything, mock.Anything).
			Return(known.Cause{}, errors.New("uh-oh"))
		service := known.NewCauseService(storage)

		_, err := service.Save(context.Background(), a.Cause().Build())

		require.Error(t, err, "expected to have failed when the underlying storage always fails")
		require.ErrorContains(t, err, "failed to store known cause:")
	})

	t.Run("on successful save returns the updated cause", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.On("Save", mock.Anything, mock.Anything).
			Return(a.Cause().Build(), nil)
		service := known.NewCauseService(storage)

		actual, err := service.Save(context.Background(), a.Cause().IsNotSaved().Build())

		require.NoError(t, err)
		require.Equal(t, a.Cause().Build(), actual)
	})
}

func TestKnownCauseService_Get(t *testing.T) {
	t.Run("wraps any storage error and returns it", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.On("Get", mock.Anything, mock.Anything).Return(known.Cause{}, errors.New("uh-oh"))
		service := known.NewCauseService(storage)

		_, actual := service.Get(context.Background(), uuid.Nil)

		require.ErrorContains(t, actual, "failed to get known cause:")
	})

	t.Run("with no errors from storage return the object as-is", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		expected := a.Cause().Build()
		storage.On("Get", mock.Anything, expected.ID).Return(expected, nil)
		service := known.NewCauseService(storage)

		actual, err := service.Get(context.Background(), expected.ID)

		require.NoError(t, err)
		require.Equal(t, expected, actual, "expected an unchanged known cause back")
	})
}

func TestKnownCauseService_All(t *testing.T) {
	t.Run("returns a wrapped error if one is returned from the storage", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.On("All", mock.Anything).Return(([]known.Cause)(nil), errors.New("uh-oh"))
		ctx := context.Background()
		service := known.NewCauseService(storage)

		_, actual := service.All(ctx)

		require.Error(t, actual, "expected an error to be returned since it's hardcoded to always give one")
		require.ErrorContains(t, actual, "unable to get all known causes from storage:")
	})

	t.Run("returns the returned object when no errors", func(t *testing.T) {
		storage := new(causeStorageMock)
		storage.Test(t)
		storage.On("All", mock.Anything).Return([]known.Cause{a.Cause().Build()}, nil)
		ctx := context.Background()
		service := known.NewCauseService(storage)

		actual, err := service.All(ctx)
		require.NoError(t, err)

		require.Equal(
			t,
			[]known.Cause{a.Cause().Build()},
			actual,
		)
	})
}
