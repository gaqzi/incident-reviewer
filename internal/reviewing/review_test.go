package reviewing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
	"github.com/gaqzi/incident-reviewer/internal/reviewing"
)

type storageMock struct {
	mock.Mock
}

func (m *storageMock) Save(ctx context.Context, review reviewing.Review) (reviewing.Review, error) {
	args := m.Called(ctx, review)
	return args.Get(0).(reviewing.Review), args.Error(1)
}

func (m *storageMock) Get(ctx context.Context, reviewID int64) (reviewing.Review, error) {
	args := m.Called(ctx, reviewID)
	return args.Get(0).(reviewing.Review), args.Error(1)
}

func (m *storageMock) All(ctx context.Context) ([]reviewing.Review, error) {
	args := m.Called(ctx)
	return args.Get(0).([]reviewing.Review), args.Error(1)
}

type causeStorageMock struct {
	mock.Mock
}

func (m *causeStorageMock) Get(ctx context.Context, id int64) (normalized.ContributingCause, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(normalized.ContributingCause), args.Error(1)
}

func TestService_Save(t *testing.T) {
	t.Run("returns the error from the underlying storage it errors", func(t *testing.T) {
		store := new(storageMock)
		store.Test(t)
		store.On("Save", mock.Anything, reviewing.Review{}).Return(reviewing.Review{}, errors.New("uh-oh"))
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		_, actual := service.Save(ctx, reviewing.Review{})

		require.Error(t, actual, "expected an error since the mock storage always fails")
		require.ErrorContains(t, actual, "failed to save review in storage:")
	})

	t.Run("returns the object from save when it saves successfully", func(t *testing.T) {
		store := new(storageMock)
		store.Test(t)
		store.On("Save", mock.Anything, reviewing.Review{}).Return(reviewing.Review{ID: 1}, nil)
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		actual, err := service.Save(ctx, reviewing.Review{})
		require.NoError(t, err)

		require.Equal(
			t,
			reviewing.Review{ID: 1},
			actual,
			"expected the returned version from storage to be returned",
		)
	})
}

func TestService_Get(t *testing.T) {
	t.Run("returns the error from the underlying storage it errors", func(t *testing.T) {
		store := new(storageMock)
		store.Test(t)
		store.On("Get", mock.Anything, int64(1)).Return(reviewing.Review{}, errors.New("uh-oh"))
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		_, actual := service.Get(ctx, 1)

		require.Error(t, actual, "expected an error since we haven't stored any reviews")
		require.ErrorContainsf(t, actual, "failed to get review:", "so we know we got the correct error")
	})

	t.Run("returns the object when there is no error", func(t *testing.T) {
		review := reviewing.Review{
			ID: 1,
		}
		store := new(storageMock)
		store.Test(t)
		store.On("Get", mock.Anything, review.ID).Return(review, nil)
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		actual, err := service.Get(ctx, review.ID)
		require.NoError(t, err)

		require.Equal(
			t,
			review,
			actual,
			"expected to have gotten back the same item as was originally saved",
		)
	})
}

func TestService_All(t *testing.T) {
	t.Run("returns the list of reviews when there is no error", func(t *testing.T) {
		store := new(storageMock)
		store.Test(t)
		store.On("All", mock.Anything).Return([]reviewing.Review(nil), nil)
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		actual, err := service.All(ctx)

		require.NoError(t, err)
		require.Empty(t, actual)
	})

	t.Run("with an error when fetching all it's wrapped and returned", func(t *testing.T) {
		store := new(storageMock)
		store.Test(t)
		store.On("All", mock.Anything).Return(([]reviewing.Review)(nil), errors.New("uh-oh"))
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		actual, err := service.All(ctx)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to get all reviews:")
		require.Nil(t, actual, "expected an empty slice returned")
	})
}

func TestService_AddContributingCause(t *testing.T) {
	t.Run("when review doesn't exist it returns the error from the storage", func(t *testing.T) {
		store := new(storageMock)
		store.Test(t)
		store.On("Get", mock.Anything, int64(1)).Return(reviewing.Review{}, errors.New("uh-oh"))
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		actual := service.AddContributingCause(ctx, 1, 1, "because")

		require.Error(t, actual, "expected an error since we haven't stored any reviews")
		require.ErrorContainsf(t, actual, "failed to get review:", "so we know we got the correct error")
	})

	t.Run("when the contributing cause isn't known return the error from it", func(t *testing.T) {
		store := new(storageMock)
		store.Test(t)
		store.On("Get", mock.Anything, int64(1)).Return(reviewing.Review{ID: 1}, nil)
		causeStore := new(causeStorageMock)
		causeStore.Test(t)
		causeStore.On("Get", mock.Anything, int64(1)).Return(normalized.ContributingCause{}, errors.New("uh-oh"))
		service := reviewing.NewService(store, causeStore)
		ctx := context.Background()

		actual := service.AddContributingCause(ctx, 1, 1, "because!")

		require.Error(t, actual, "expected an error when invalid cause provided")
		require.ErrorContains(t, actual, "failed to get contributing cause:")
	})

	t.Run("when both review and contributing cause are known bind it", func(t *testing.T) {
		store := new(storageMock)
		store.Test(t)
		store.On("Get", mock.Anything, int64(1)).Return(reviewing.Review{ID: 1}, nil)
		causeStore := new(causeStorageMock)
		causeStore.Test(t)
		causeStore.On("Get", mock.Anything, int64(1)).Return(normalized.ContributingCause{ID: 1}, nil)
		storedReview := reviewing.Review{
			ID: 1,
			ContributingCauses: []reviewing.ReviewCause{{ // make sure we create the ReviewCause correctly and attach it
				Cause: normalized.ContributingCause{ID: 1},
				Why:   "because",
			}},
		}
		store.On("Save", mock.Anything, storedReview).Return(storedReview, nil)
		service := reviewing.NewService(store, causeStore)
		ctx := context.Background()

		actual := service.AddContributingCause(ctx, 1, 1, "because")
		require.NoError(t, actual, "expected to have bound the cause to the review successfully")
	})
}

func TestReview_Update(t *testing.T) {
	t.Run("an update with no changes doesn't modify the object", func(t *testing.T) {
		orig := reviewing.Review{ID: 1}
		upd := reviewing.Review{ID: 1}

		actual := orig.Update(upd)

		require.Equal(t, reviewing.Review{ID: 1}, actual, "expected orig to not have changed since all fields are the same")
	})

	t.Run("an update to an allowed field updates the original object", func(t *testing.T) {
		orig := reviewing.Review{ID: 1}
		upd := reviewing.Review{ID: 2, URL: "http://example.com/"}

		actual := orig.Update(upd)

		require.Equal(
			t,
			reviewing.Review{ID: 1, URL: "http://example.com/"},
			actual,
			"expected to have added the URL into the original object",
		)
	})

	for _, tc := range []struct {
		name     string
		upd      reviewing.Review
		expected reviewing.Review
	}{
		{
			"URL",
			reviewing.Review{URL: "http://example.com/"},
			reviewing.Review{ID: 1, URL: "http://example.com/"},
		},
		{
			"Title",
			reviewing.Review{Title: "example"},
			reviewing.Review{ID: 1, Title: "example"},
		},
		{
			"Description",
			reviewing.Review{Description: "example"},
			reviewing.Review{ID: 1, Description: "example"},
		},
		{
			"Impact",
			reviewing.Review{Impact: "example"},
			reviewing.Review{ID: 1, Impact: "example"},
		},
		{
			"Where",
			reviewing.Review{Where: "example"},
			reviewing.Review{ID: 1, Where: "example"},
		},
		{
			"ReportProximalCause",
			reviewing.Review{ReportProximalCause: "example"},
			reviewing.Review{ID: 1, ReportProximalCause: "example"},
		},
		{
			"ReportTrigger: ",
			reviewing.Review{ReportTrigger: "example"},
			reviewing.Review{ID: 1, ReportTrigger: "example"},
		},
	} {
		t.Run("updates field: "+tc.name, func(t *testing.T) {
			orig := reviewing.Review{ID: 1}

			actual := orig.Update(tc.upd)

			require.Equal(t, tc.expected, actual)
		})
	}
}
