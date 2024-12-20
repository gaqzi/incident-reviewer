package reviewing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
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

func (m *storageMock) Get(ctx context.Context, reviewID uuid.UUID) (reviewing.Review, error) {
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

func (m *causeStorageMock) Get(ctx context.Context, id uuid.UUID) (contributing.Cause, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(contributing.Cause), args.Error(1)
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
		id := uuid.Must(uuid.NewV7())
		store.On("Get", mock.Anything, id).Return(reviewing.Review{}, errors.New("uh-oh"))
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		_, actual := service.Get(ctx, id)

		require.Error(t, actual, "expected an error since we haven't stored any reviews")
		require.ErrorContainsf(t, actual, "failed to get review:", "so we know we got the correct error")
	})

	t.Run("returns the object when there is no error", func(t *testing.T) {
		expected := a.Review().Build()
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
		id := uuid.Must(uuid.NewV7())
		store := new(storageMock)
		store.Test(t)
		store.On("Get", mock.Anything, id).Return(reviewing.Review{}, errors.New("uh-oh"))
		service := reviewing.NewService(store, nil)
		ctx := context.Background()

		actual := service.AddContributingCause(ctx, id, uuid.Nil, reviewing.ReviewCause{Why: "because", IsProximalCause: false})

		require.Error(t, actual, "expected an error since we haven't stored any reviews")
		require.ErrorContainsf(t, actual, "failed to get review:", "so we know we got the correct error")
	})

	t.Run("when the contributing cause isn't known return the error from it", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		store := new(storageMock)
		store.Test(t)
		store.On("Get", mock.Anything, id).Return(reviewing.Review{ID: id}, nil)
		causeStore := new(causeStorageMock)
		causeStore.Test(t)
		causeStore.On("Get", mock.Anything, uuid.Nil).Return(contributing.Cause{}, errors.New("uh-oh"))
		service := reviewing.NewService(store, causeStore)
		ctx := context.Background()

		actual := service.AddContributingCause(ctx, id, uuid.Nil, reviewing.ReviewCause{Why: "because!", IsProximalCause: false})

		require.Error(t, actual, "expected an error when invalid cause provided")
		require.ErrorContains(t, actual, "failed to get contributing cause:")
	})

	t.Run("when both review and contributing cause are known bind it", func(t *testing.T) {
		store := new(storageMock)
		store.Test(t)
		review := a.Review().IsValid().IsSaved().Build()
		store.On("Get", mock.Anything, review.ID).Return(review, nil)
		causeStore := new(causeStorageMock)
		causeStore.Test(t)
		cause := a.ContributingCause().WithID(uuid.Nil).Build() // Intentional set the nil UUID to make sure we look up what we're saying and not binding otherwise
		causeStore.On("Get", mock.Anything, uuid.Nil).Return(cause, nil)
		storedReview := review
		storedReview.ContributingCauses = append(storedReview.ContributingCauses, reviewing.ReviewCause{ // make sure we create the ReviewCause correctly and attach it
			Cause:           cause,
			Why:             "because",
			IsProximalCause: false,
		})
		store.On("Save", mock.Anything, mock.IsType(reviewing.Review{})).Return(storedReview, nil)
		service := reviewing.NewService(store, causeStore)
		ctx := context.Background()

		actual := service.AddContributingCause(ctx, review.ID, uuid.Nil, reviewing.ReviewCause{
			Cause:           a.ContributingCause().Build(), // pass in a cause but only bind by the ID passed in to look up
			Why:             "because",
			IsProximalCause: false,
		})
		require.NoError(t, actual, "expected to have bound the cause to the review successfully")
	})

	t.Run("ensure we add the contributing cause through the aggregate root", func(t *testing.T) {
		// This test is here just because I'm mixing behavior and collaboration in the service,
		// and because I can't stub out the Review for tests. So make sure _one_ behavior from the validation is here.
		ctx := context.Background()
		cause := reviewing.ReviewCause{Cause: a.ContributingCause().Build(), Why: "because"}
		store := new(storageMock)
		store.Test(t)
		// First time, no causes added
		store.On("Get", mock.Anything, mock.Anything).Return(a.Review().Build(), nil).Once()
		// Second time, the cause has been saved, urgh, this test is messy AF
		store.On("Get", mock.Anything, mock.Anything).
			Return(
				a.Review().
					WithContributingCause(cause).
					Build(),
				nil,
			).
			Once()
		store.On("Save", mock.Anything, mock.Anything).Once().Return(a.Review().Build(), nil)
		causeStore := new(causeStorageMock)
		causeStore.Test(t)
		causeStore.On("Get", mock.Anything, mock.Anything).Return(a.ContributingCause().Build(), nil)
		service := reviewing.NewService(store, causeStore)

		actual := service.AddContributingCause(ctx, uuid.Nil, uuid.Nil, cause)
		require.NoError(t, actual)

		actual = service.AddContributingCause(ctx, uuid.Nil, uuid.Nil, cause)
		require.ErrorContains(t, actual, "failed to add contributing cause:")
	})
}

func TestReview_Update(t *testing.T) {
	t.Run("an update with no changes doesn't modify the object", func(t *testing.T) {
		orig := a.Review().Build()
		upd := a.Review().Build()

		actual := orig.Update(upd)

		require.Equal(t, a.Review().Build(), actual, "expected orig to not have changed since all fields are the same")
	})

	t.Run("an update to an allowed field updates the original object", func(t *testing.T) {
		orig := a.Review().Build()
		upd := a.Review().WithID(uuid.Must(uuid.NewV7())).WithURL("http://example.com/").Build()

		actual := orig.Update(upd)

		require.Equal(
			t,
			a.Review().WithURL("http://example.com/").Build(),
			actual,
			"expected to have added the URL into the original object",
		)
	})

	id := uuid.Must(uuid.NewV7())
	for _, tc := range []struct {
		name     string
		upd      reviewing.Review
		expected reviewing.Review
	}{
		{
			"URL",
			reviewing.Review{URL: "http://example.com/"},
			reviewing.Review{ID: id, URL: "http://example.com/"},
		},
		{
			"Title",
			reviewing.Review{Title: "example"},
			reviewing.Review{ID: id, Title: "example"},
		},
		{
			"Description",
			reviewing.Review{Description: "example"},
			reviewing.Review{ID: id, Description: "example"},
		},
		{
			"Impact",
			reviewing.Review{Impact: "example"},
			reviewing.Review{ID: id, Impact: "example"},
		},
		{
			"Where",
			reviewing.Review{Where: "example"},
			reviewing.Review{ID: id, Where: "example"},
		},
		{
			"ReportProximalCause",
			reviewing.Review{ReportProximalCause: "example"},
			reviewing.Review{ID: id, ReportProximalCause: "example"},
		},
		{
			"ReportTrigger: ",
			reviewing.Review{ReportTrigger: "example"},
			reviewing.Review{ID: id, ReportTrigger: "example"},
		},
	} {
		t.Run("updates field: "+tc.name, func(t *testing.T) {
			orig := reviewing.Review{ID: id}

			actual := orig.Update(tc.upd)

			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestReview_AddContributingCause(t *testing.T) {
	t.Run("adds the contributing cause to the list of contributing causes", func(t *testing.T) {
		review := a.Review().Build()
		cause := a.ContributingCause().Build()
		reviewCause := reviewing.ReviewCause{Cause: cause, Why: "because", IsProximalCause: false}

		review, err := review.AddContributingCause(reviewCause)

		require.NoError(t, err)
		require.Equal(t, []reviewing.ReviewCause{reviewCause}, review.ContributingCauses, "expected the new bonud cause to be the only one in the list")
	})

	t.Run("returns an error when the same cause is added with the same why", func(t *testing.T) {
		for _, tc := range []struct {
			description string
			why         string
		}{
			{"literally the same", "because"},
			{"why with lots of surrounding spaces", "\t because\n \t"},
			{"why with different cases", "bEcAuSe"},
		} {
			t.Run(tc.description, func(t *testing.T) {
				review := a.Review().Build()
				cause := a.ContributingCause().Build()
				reviewCause := reviewing.ReviewCause{Cause: cause, Why: "because", IsProximalCause: false}

				review, err := review.AddContributingCause(reviewCause)
				require.NoError(t, err)

				reviewCause.Why = tc.why
				review, err = review.AddContributingCause(reviewCause)
				require.Error(t, err)
				require.ErrorContains(t, err, "cannot bind contributing cause with the same why: "+tc.why)
			})
		}
	})

	t.Run("when setting the proximal cause sets all previously stored as not proximal", func(t *testing.T) {
		review := a.Review().Build()
		cause := a.ContributingCause().Build()
		cause2 := a.ContributingCause().WithID(uuid.Nil).Build()

		review, err := review.AddContributingCause(reviewing.ReviewCause{Cause: cause, Why: "because", IsProximalCause: true})
		require.NoError(t, err)
		require.True(t, review.ContributingCauses[0].IsProximalCause)

		review, err = review.AddContributingCause(reviewing.ReviewCause{Cause: cause2, Why: "why not?", IsProximalCause: true})
		require.NoError(t, err)

		proximalCauseMap := map[string]bool{}
		for _, cause := range review.ContributingCauses {
			proximalCauseMap[cause.Cause.ID.String()] = cause.IsProximalCause
		}
		require.Equal(
			t,
			map[string]bool{cause.ID.String(): false, cause2.ID.String(): true},
			proximalCauseMap,
			"expected the second cause to be marked as proximal and the first to have been unmarked",
		)
	})
}
