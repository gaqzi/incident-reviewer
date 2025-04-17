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

type triggerStorageMock struct {
	mock.Mock
}

func (m *triggerStorageMock) Get(ctx context.Context, id uuid.UUID) (known.Trigger, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(known.Trigger), args.Error(1)
}

func (m *triggerStorageMock) Save(ctx context.Context, trigger known.Trigger) (known.Trigger, error) {
	args := m.Called(ctx, trigger)
	return args.Get(0).(known.Trigger), args.Error(1)
}

func (m *triggerStorageMock) All(ctx context.Context) ([]known.Trigger, error) {
	args := m.Called(ctx)
	return args.Get(0).([]known.Trigger), args.Error(1)
}

func TestKnownTriggerService_Save(t *testing.T) {
	t.Run("sets the Created and Updated at when they're not set", func(t *testing.T) {
		storage := new(triggerStorageMock)
		storage.Test(t)
		storage.
			On("Save", mock.Anything, mock.MatchedBy(func(c known.Trigger) bool {
				return !c.CreatedAt.IsZero() && !c.UpdatedAt.IsZero() && c.CreatedAt.Equal(c.UpdatedAt)
			})).
			Return(known.Trigger{}, nil)
		service := known.NewTriggerService(storage)

		_, err := service.Save(context.Background(), a.Trigger().IsNotSaved().Build())

		require.NoError(t, err)
	})

	t.Run("sets the UpdatedAt when updating a previously saved item", func(t *testing.T) {
		storage := new(triggerStorageMock)
		storage.Test(t)
		storage.
			On("Save", mock.Anything, mock.MatchedBy(func(c known.Trigger) bool {
				return !c.UpdatedAt.IsZero() && c.UpdatedAt.After(c.CreatedAt)
			})).
			Return(known.Trigger{}, nil)
		service := known.NewTriggerService(storage)

		_, err := service.Save(context.Background(), a.Trigger().IsSaved().Build())

		require.NoError(t, err)
	})

	t.Run("validate the Trigger object before saving", func(t *testing.T) {
		service := known.NewTriggerService(nil)

		_, actual := service.Save(context.Background(), known.Trigger{})

		require.Error(t, actual)
		require.ErrorContains(t, actual, "failed to validate trigger:")
		var errs validator.ValidationErrors
		require.ErrorAs(t, actual, &errs)
		require.GreaterOrEqual(t, len(errs), 3, "expected to have at minimum 3 errors for the required fields")
	})

	t.Run("wraps any error from the store and returns it", func(t *testing.T) {
		storage := new(triggerStorageMock)
		storage.Test(t)
		storage.On("Save", mock.Anything, mock.Anything).
			Return(known.Trigger{}, errors.New("uh-oh"))
		service := known.NewTriggerService(storage)

		_, err := service.Save(context.Background(), a.Trigger().Build())

		require.Error(t, err, "expected to have failed when the underlying storage always fails")
		require.ErrorContains(t, err, "failed to store known trigger:")
	})

	t.Run("on successful save returns the updated trigger", func(t *testing.T) {
		storage := new(triggerStorageMock)
		storage.Test(t)
		storage.On("Save", mock.Anything, mock.Anything).
			Return(a.Trigger().Build(), nil)
		service := known.NewTriggerService(storage)

		actual, err := service.Save(context.Background(), a.Trigger().IsNotSaved().Build())

		require.NoError(t, err)
		require.Equal(t, a.Trigger().Build(), actual)
	})
}

func TestKnownTriggerService_Get(t *testing.T) {
	t.Run("wraps any storage error and returns it", func(t *testing.T) {
		storage := new(triggerStorageMock)
		storage.Test(t)
		storage.On("Get", mock.Anything, mock.Anything).Return(known.Trigger{}, errors.New("uh-oh"))
		service := known.NewTriggerService(storage)

		_, actual := service.Get(context.Background(), uuid.Nil)

		require.ErrorContains(t, actual, "failed to get trigger:")
	})

	t.Run("with no errors from storage return the object as-is", func(t *testing.T) {
		storage := new(triggerStorageMock)
		storage.Test(t)
		expected := a.Trigger().Build()
		storage.On("Get", mock.Anything, expected.ID).Return(expected, nil)
		service := known.NewTriggerService(storage)

		actual, err := service.Get(context.Background(), expected.ID)

		require.NoError(t, err)
		require.Equal(t, expected, actual, "expected an unchanged known trigger back")
	})
}

func TestKnownTriggerService_All(t *testing.T) {
	t.Run("returns a wrapped error if one is returned from the storage", func(t *testing.T) {
		storage := new(triggerStorageMock)
		storage.Test(t)
		storage.On("All", mock.Anything).Return(([]known.Trigger)(nil), errors.New("uh-oh"))
		ctx := context.Background()
		service := known.NewTriggerService(storage)

		_, actual := service.All(ctx)

		require.Error(t, actual, "expected an error to be returned since it's hardcoded to always give one")
		require.ErrorContains(t, actual, "unable to get all triggers from storage:")
	})

	t.Run("returns the returned object when no errors", func(t *testing.T) {
		storage := new(triggerStorageMock)
		storage.Test(t)
		storage.On("All", mock.Anything).Return([]known.Trigger{a.Trigger().Build()}, nil)
		ctx := context.Background()
		service := known.NewTriggerService(storage)

		actual, err := service.All(ctx)
		require.NoError(t, err)

		require.Equal(
			t,
			[]known.Trigger{a.Trigger().Build()},
			actual,
		)
	})
}
