package normalized_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
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
