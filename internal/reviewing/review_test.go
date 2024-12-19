package reviewing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
	"github.com/gaqzi/incident-reviewer/internal/reviewing"
	"github.com/gaqzi/incident-reviewer/test/a"
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
		review := a.Review().IsValid().Build()
		store.On("Save", mock.Anything, mock.IsType(reviewing.Review{})).Return(reviewing.Review{}, errors.New("uh-oh"))
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		_, actual := service.Save(ctx, review)

		require.Error(t, actual, "expected an error since the mock storage always fails")
		require.ErrorContains(t, actual, "failed to save review in storage:")
	})

	t.Run("returns the object from save when it saves successfully", func(t *testing.T) {
		store := new(storageMock)
		store.Test(t)
		store.
			On("Save", mock.Anything, mock.IsType(reviewing.Review{})).
			Return(a.Review().Build(), nil)
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		actual, err := service.Save(ctx, a.Review().IsNotSaved().Build())
		require.NoError(t, err)

		require.Equal(
			t,
			a.Review().Build(),
			actual,
			"expected the returned version from storage to be returned",
		)
	})

	t.Run("validates the Review and returns an error when validation fails", func(t *testing.T) {
		service := reviewing.NewService(nil, nil)
		ctx := context.Background()

		_, actual := service.Save(ctx, reviewing.Review{})

		var errs validator.ValidationErrors
		require.ErrorAs(t, actual, &errs, "expected an empty review to be invalid and have the invalid fields returned")
		require.ErrorContains(t, actual, "failed to validate review:")
		require.GreaterOrEqual(t, len(errs), 8, "expected at minimum 8 errors to match the fields at the time of writing")
	})

	t.Run("it calls Review.updateTimestamps() to set the times when saving", func(t *testing.T) {
		t.Run("on a new object it sets created at and updated at", func(t *testing.T) {
			store := new(storageMock)
			store.Test(t)
			store.
				On("Save", mock.Anything, mock.MatchedBy(func(r reviewing.Review) bool {
					// Ensure that we've set Created and Updated At and that they're the same
					// The exact values aren't important for our test, just that they've been set.
					//
					// XXX: Is this a bad design since I'm testing that some collaboration happened,
					// but I don't know the full values because it's on the aggregate root? This is
					// the best I could come up with for now… 😅
					return !r.CreatedAt.IsZero() && !r.UpdatedAt.IsZero() && r.CreatedAt.Equal(r.UpdatedAt)
				})).
				Return(a.Review().Build(), nil)
			service := reviewing.NewService(store, nil)
			ctx := context.Background()

			actual, err := service.Save(ctx, a.Review().IsNotSaved().Build())

			require.NoError(t, err)
			require.Equal(t, a.Review().Build(), actual, "expected to have gotten back the saved object with no further changes")
		})

		t.Run("on a previously saved object (i.e. CreatedAt set) it only updated UpdatedAt", func(t *testing.T) {
			store := new(storageMock)
			store.Test(t)
			store.
				On("Save", mock.Anything, mock.MatchedBy(func(r reviewing.Review) bool {
					// Ensure CreatedAt and UpdatedAt are different
					return !r.CreatedAt.Equal(r.UpdatedAt)
				})).
				Return(a.Review().Build(), nil)
			service := reviewing.NewService(store, nil)
			ctx := context.Background()

			_, err := service.Save(ctx, a.Review().IsSaved().Build())

			require.NoError(t, err)
		})
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
		expected := reviewing.Review{ID: 1}
		store := new(storageMock)
		store.Test(t)
		store.On("Get", mock.Anything, expected.ID).Return(expected, nil)
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		actual, err := service.Get(ctx, expected.ID)
		require.NoError(t, err)

		require.Equal(
			t,
			expected,
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
		review := a.Review().IsValid().IsSaved().Build()
		store.On("Get", mock.Anything, int64(1)).Return(review, nil)
		causeStore := new(causeStorageMock)
		causeStore.Test(t)
		causeStore.On("Get", mock.Anything, int64(1)).Return(normalized.ContributingCause{ID: 1}, nil)
		storedReview := review
		storedReview.ContributingCauses = append(storedReview.ContributingCauses, reviewing.ReviewCause{ // make sure we create the ReviewCause correctly and attach it
			Cause: normalized.ContributingCause{ID: 1},
			Why:   "because",
		})
		store.On("Save", mock.Anything, mock.IsType(reviewing.Review{})).Return(storedReview, nil)
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
