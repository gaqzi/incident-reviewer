package reviewing_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
	"github.com/gaqzi/incident-reviewer/internal/platform/action"
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

type builderService struct {
	reviewStorage *storageMock
	causeStorage  *causeStorageMock
	actionMapper  *action.Mapper
}

func newService() builderService {
	return builderService{
		reviewStorage: new(storageMock),
		causeStorage:  new(causeStorageMock),
		actionMapper:  &action.Mapper{},
	}
}

func (b builderService) Build(t *testing.T) *reviewing.Service {
	t.Helper()
	rs := b.reviewStorage
	rs.Test(t)
	cs := b.causeStorage
	cs.Test(t)
	return reviewing.NewService(rs, cs, reviewing.WithActionMapper(b.actionMapper))
}

func (b builderService) getReview(r reviewing.Review) builderService {
	b.reviewStorage.On("Get", mock.Anything, r.ID).Return(r, nil)

	return b
}

func (b builderService) getReviewFail(err ...error) builderService {
	if err == nil {
		err = append(err, errors.New("uh-oh"))
	}

	b.reviewStorage.On("Get", mock.Anything, mock.Anything).Return(reviewing.Review{}, err[0])

	return b
}

func (b builderService) saveAction(er reviewing.Review) builderService {
	b.actionMapper.Add("Save", func(_ context.Context, r reviewing.Review) (reviewing.Review, error) {
		if !reflect.DeepEqual(er, r) {
			return reviewing.Review{}, errors.New("the passed in values don't match the expected values")
		}

		return r, nil
	})

	return b
}

func (b builderService) saveActionFail(err ...error) builderService {
	if err == nil {
		err = append(err, errors.New("uh-oh"))
	}

	b.actionMapper.Add("Save", func(_ context.Context, _ reviewing.Review) (reviewing.Review, error) {
		return reviewing.Review{}, err[0]
	})

	return b
}

func (b builderService) saveReviewFail() builderService {
	b.reviewStorage.On("Save", mock.Anything, mock.IsType(reviewing.Review{})).
		Return(reviewing.Review{}, errors.New("uh-oh"))

	return b
}

func (b builderService) saveReview(r reviewing.Review) builderService {
	b.reviewStorage.On("Save", mock.Anything, mock.IsType(reviewing.Review{})).
		Return(r, nil)

	return b
}

func (b builderService) allReviews(rs []reviewing.Review) builderService {
	b.reviewStorage.On("All", mock.Anything).Return(rs, nil)

	return b
}

func (b builderService) allReviewsFail(err ...error) builderService {
	if err == nil {
		err = append(err, errors.New("uh-oh"))
	}

	b.reviewStorage.On("All", mock.Anything).Return([]reviewing.Review(nil), err[0])

	return b
}

func (b builderService) getCauseFail(err ...error) builderService {
	if err == nil {
		err = append(err, errors.New("uh-oh"))
	}

	b.causeStorage.On("Get", mock.Anything, mock.Anything).Return(contributing.Cause{}, err[0])

	return b
}

func (b builderService) getCause(cause contributing.Cause) builderService {
	b.causeStorage.On("Get", mock.Anything, cause.ID).Return(cause, nil)

	return b
}

func (b builderService) addContributingCauseActionFail(err ...error) builderService {
	if err == nil {
		err = append(err, errors.New("uh-oh"))
	}

	b.actionMapper.Add("AddContributingCause", func(_ reviewing.Review, _ contributing.Cause, _ reviewing.ReviewCause) (reviewing.Review, error) {
		return reviewing.Review{}, err[0]
	})

	return b
}

func (b builderService) addContributingCauseAction(er reviewing.Review, ec contributing.Cause, erc reviewing.ReviewCause) builderService {
	b.actionMapper.Add("AddContributingCause", func(r reviewing.Review, c contributing.Cause, rc reviewing.ReviewCause) (reviewing.Review, error) {
		if !reflect.DeepEqual(er, r) ||
			!reflect.DeepEqual(ec, c) ||
			!reflect.DeepEqual(erc, rc) {
			return reviewing.Review{}, errors.New("the passed in values don't match the expected values")
		}

		return r, nil
	})

	return b
}

func (b builderService) updateBoundContributingCauseActionFail() builderService {
	b.actionMapper.Add("UpdateBoundContributingCause", func(_ reviewing.Review, _ reviewing.ReviewCause) (reviewing.Review, error) {
		return reviewing.Review{}, errors.New("uh-oh")
	})

	return b
}

func (b builderService) updateBoundContributingCauseAction(er reviewing.Review, ec reviewing.ReviewCause) builderService {
	b.actionMapper.Add("UpdateBoundContributingCause", func(r reviewing.Review, c reviewing.ReviewCause) (reviewing.Review, error) {
		if !reflect.DeepEqual(er, r) ||
			!reflect.DeepEqual(ec, c) {
			return reviewing.Review{}, errors.New("the passed in values don't match the expected values")
		}

		return r, nil
	})

	return b
}

func TestService_Save(t *testing.T) {
	t.Run("wraps any error from collaborating with action mapper", func(t *testing.T) {
		service := newService().
			saveActionFail().
			Build(t)

		_, actual := service.Save(context.Background(), reviewing.Review{})

		require.ErrorContains(t, actual, "pre-save action failed:")
	})

	t.Run("returns the error from the underlying storage when it errors", func(t *testing.T) {
		review := a.Review().IsValid().Build()
		service := newService().
			saveAction((func(r reviewing.Review) reviewing.Review {
				return r
			})(review)).
			saveReviewFail().
			Build(t)

		_, actual := service.Save(context.Background(), review)

		require.Error(t, actual, "expected an error since the mock storage always fails")
		require.ErrorContains(t, actual, "failed to save review in storage:")
	})

	t.Run("returns the object from save when it saves successfully", func(t *testing.T) {
		review := a.Review().IsNotSaved().Build()
		service := newService().
			saveAction(review).
			saveReview(a.Review().Build()).
			Build(t)

		actual, err := service.Save(context.Background(), review)
		require.NoError(t, err)

		require.Equal(
			t,
			a.Review().Build(),
			actual,
			"expected the returned version from storage to be returned",
		)
	})
}

func TestService_Get(t *testing.T) {
	t.Run("returns the error from the underlying storage it errors", func(t *testing.T) {
		service := newService().
			getReviewFail(errors.New("uh-oh")).
			Build(t)

		_, actual := service.Get(context.Background(), a.UUID())

		require.Error(t, actual, "expected an error since we haven't stored any reviews")
		require.ErrorContainsf(t, actual, "failed to get review:", "so we know we got the correct error")
	})

	t.Run("returns the object when there is no error", func(t *testing.T) {
		expected := a.Review().Build()
		service := newService().
			getReview(expected).
			Build(t)

		actual, err := service.Get(context.Background(), expected.ID)
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
		service := newService().
			allReviews([]reviewing.Review(nil)).
			Build(t)

		actual, err := service.All(context.Background())

		require.NoError(t, err)
		require.Empty(t, actual)
	})

	t.Run("with an error when fetching all it's wrapped and returned", func(t *testing.T) {
		service := newService().
			allReviewsFail().
			Build(t)

		actual, err := service.All(context.Background())

		require.ErrorContains(t, err, "failed to get all reviews:")
		require.Nil(t, actual, "expected an empty slice returned")
	})
}

func TestService_AddContributingCause(t *testing.T) {
	t.Run("when review doesn't exist it returns the error from the storage", func(t *testing.T) {
		service := newService().
			getReviewFail().
			Build(t)
		ctx := context.Background()

		actual := service.AddContributingCause(ctx, uuid.Nil, uuid.Nil, a.ReviewCause().Build())

		require.Error(t, actual, "expected an error since we haven't stored any reviews")
		require.ErrorContainsf(t, actual, "failed to get review:", "so we know we got the correct error")
	})

	t.Run("when the contributing cause isn't known return the error from it", func(t *testing.T) {
		review := a.Review().Build()
		service := newService().
			getReview(review).
			getCauseFail().
			Build(t)

		actual := service.AddContributingCause(
			context.Background(),
			review.ID,
			uuid.Nil,
			a.ReviewCause().Build(),
		)

		require.ErrorContains(t, actual, "failed to get contributing cause:")
	})

	t.Run("it returns any errors when adding the cause to the review", func(t *testing.T) {
		review := a.Review().Build()
		reviewCause := a.ReviewCause().Build()
		service := newService().
			getReview(review).
			getCause(reviewCause.Cause).
			addContributingCauseActionFail().
			Build(t)

		actual := service.AddContributingCause(context.Background(), review.ID, reviewCause.Cause.ID, reviewCause)

		require.ErrorContains(t, actual, "failed to add contributing cause to review:")
	})

	t.Run("when both review and contributing cause are known bind it", func(t *testing.T) {
		review := a.Review().Build()
		cause := a.ContributingCause().Build()
		reviewCause := a.ReviewCause().WithCause(cause).Build()
		service := newService().
			getReview(review).
			getCause(reviewCause.Cause).
			saveAction(review).
			addContributingCauseAction(review, cause, reviewCause).
			saveReview(review).
			Build(t)
		ctx := context.Background()

		actual := service.AddContributingCause(ctx, review.ID, cause.ID, reviewCause)
		require.NoError(t, actual, "expected to have bound the cause to the review successfully")
	})
}

func TestService_GetBoundContributingCause(t *testing.T) {
	t.Run("when the review doesn't exist it returns an error", func(t *testing.T) {
		service := newService().
			getReviewFail().
			Build(t)

		_, actual := service.GetBoundContributingCause(context.Background(), uuid.Nil, uuid.Nil)

		require.ErrorContains(t, actual, "review with that id not found to relate bound contributing cause:")
	})

	t.Run("when the contributing cause hasn't been bound it returns an error", func(t *testing.T) {
		review := a.Review().Build()
		service := newService().
			getReview(review).
			Build(t)

		_, actual := service.GetBoundContributingCause(context.Background(), review.ID, a.UUID())

		require.ErrorContains(t, actual, "review doesn't have that contributing cause bound:")
	})

	t.Run("when it's found it returns the reviewing.ReviewCause", func(t *testing.T) {
		boundCause := a.ReviewCause().Build()
		store := new(storageMock)
		store.Test(t)
		review := a.Review().WithContributingCause(boundCause).Build()
		service := newService().
			getReview(review).
			Build(t)

		actual, err := service.GetBoundContributingCause(context.Background(), review.ID, boundCause.ID)

		require.NoError(t, err)
		require.Equal(t, boundCause, actual, "expected the matching cause added into the reviewing.Review to be returned")
	})
}

func TestService_UpdateBoundContributingCause(t *testing.T) {
	t.Run("when the review doesn't exist it returns an error", func(t *testing.T) {
		service := newService().
			getReviewFail().
			Build(t)

		_, err := service.UpdateBoundContributingCause(context.Background(), a.UUID(), reviewing.ReviewCause{})

		require.ErrorContains(t, err, "failed to get review:")
	})

	t.Run("when the new boundCause.Cause isn't found it returns an error", func(t *testing.T) {
		review := a.Review().Build()
		reviewCause := a.ReviewCause().Build()
		service := newService().
			getReview(review).
			getCauseFail().
			Build(t)

		_, err := service.UpdateBoundContributingCause(context.Background(), review.ID, reviewCause)

		require.ErrorContains(t, err, "failed to get contributing cause:")
	})

	t.Run("when there is an error updating it returns an error", func(t *testing.T) {
		review := a.Review().WithContributingCause().Build()
		cause := review.ContributingCauses[0].Cause
		updatedCause := review.ContributingCauses[0]
		updatedCause.Why = "updated cause"
		service := newService().
			getReview(review).
			getCause(cause).
			updateBoundContributingCauseActionFail().
			Build(t)

		_, err := service.UpdateBoundContributingCause(context.Background(), review.ID, updatedCause)

		require.ErrorContains(t, err, "action to update bound contributing cause failed:")
	})

	t.Run("when the update is successful it returns the updated reviewing.ReviewCause", func(t *testing.T) {
		review := a.Review().WithContributingCause().Build()
		updatedCause := a.ReviewCause().WithWhy("updated cause").Build()
		service := newService().
			getReview(review).
			getCause(review.ContributingCauses[0].Cause).
			updateBoundContributingCauseAction(review, updatedCause). // doesn't update anything
			saveAction(review). // because it didn't update anything we just get the original passed in again
			saveReview((func(r reviewing.Review) reviewing.Review { // return something different to show that we're returning whatever is successfully saved
				r.ContributingCauses[0] = updatedCause
				return r
			})(review)).
			Build(t)

		actual, err := service.UpdateBoundContributingCause(context.Background(), review.ID, updatedCause)

		require.NoError(t, err)
		require.Equal(t, updatedCause, actual)
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
		reviewCause := a.ReviewCause().Build()

		review, err := review.AddContributingCause(reviewCause)

		require.NoError(t, err)
		require.Equal(t, []reviewing.ReviewCause{reviewCause}, review.ContributingCauses, "expected the new bound cause to be the only one in the list")
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
				_, err = review.AddContributingCause(reviewCause)
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

func TestReview_UpdateBoundContributingCause(t *testing.T) {
	t.Run("when the updated cause isn't already bound it returns an error", func(t *testing.T) {
		review := a.Review().Build()
		updatedCause := reviewing.ReviewCause{Cause: a.ContributingCause().Build(), Why: "new cause"}

		_, err := review.UpdateBoundContributingCause(updatedCause)

		require.Error(t, err)
		require.ErrorContains(t, err, "cannot update contributing cause that isn't already bound")
	})

	t.Run("when the updated cause is updated it does it by reusing the logic of AddContributingCause", func(t *testing.T) {
		firstCause := a.ReviewCause().Build()
		secondCause := a.ReviewCause().
			WithIsProximalCause(true).
			WithCause(a.ContributingCause().WithID(a.UUID()).WithName("Something different").Build()).
			WithID(a.UUID()).
			Build()
		review := a.Review().
			WithContributingCause(firstCause).
			WithContributingCause(secondCause).
			Build()
		updatedCause := a.ReviewCause().WithWhy("updated cause").WithIsProximalCause(true).Build()

		actual, err := review.UpdateBoundContributingCause(updatedCause)

		require.NoError(t, err)
		secondUpdated := secondCause
		secondUpdated.IsProximalCause = false
		require.Equal(
			t,
			[]reviewing.ReviewCause{secondUpdated, updatedCause},
			actual.ContributingCauses,
			"expected the proximal cause to have been removed from the second cause",
		)
	})
}
