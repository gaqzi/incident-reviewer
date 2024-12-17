package reviewing

import (
	"context"
	"fmt"
	"time"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
)

type Review struct {
	ID                  int64
	URL                 string `validate:"required,http_url"`
	Title               string `validate:"required"`
	Description         string `validate:"required"`
	Impact              string `validate:"required"`
	Where               string `validate:"required"`
	ReportProximalCause string `validate:"required"`
	ReportTrigger       string `validate:"required"`

	ContributingCauses []ReviewCause

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Update takes the values from the passed in review and sets the fields that are allowed for mass changes.
// This helper exists to ensure that if we are doing "blind updates" we
// don't update fields we haven't intended to. Overriding the created time, the ID, and the like.
func (r Review) Update(o Review) Review {
	r.URL = o.URL
	r.Title = o.Title
	r.Description = o.Description
	r.Impact = o.Impact
	r.Where = o.Where
	r.ReportProximalCause = o.ReportProximalCause
	r.ReportTrigger = o.ReportTrigger

	return r
}

type ReviewCause struct {
	Cause normalized.ContributingCause `validate:"required"`
	Why   string                       `validate:"required"`
}

type causeStore interface {
	Get(ctx context.Context, ID int64) (normalized.ContributingCause, error)
}

type Service struct {
	reviewStore Storage
	causeStore  causeStore
}

func NewService(reviewStore Storage, causeStore causeStore) *Service {
	return &Service{
		reviewStore: reviewStore,
		causeStore:  causeStore,
	}
}

func (s *Service) Save(ctx context.Context, review Review) (Review, error) {
	review, err := s.reviewStore.Save(ctx, review)
	if err != nil {
		return Review{}, fmt.Errorf("failed to save review in storage: %w", err)
	}

	return review, nil
}

func (s *Service) Get(ctx context.Context, reviewID int64) (Review, error) {
	review, err := s.reviewStore.Get(ctx, reviewID)
	if err != nil {
		return Review{}, fmt.Errorf("failed to get review: %w", err)
	}

	return review, nil
}

func (s *Service) All(ctx context.Context) ([]Review, error) {
	ret, err := s.reviewStore.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all reviews: %w", err)
	}

	return ret, nil
}

func (s *Service) AddContributingCause(ctx context.Context, reviewID int64, causeID int64, why string) error {
	review, err := s.reviewStore.Get(ctx, reviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}

	cause, err := s.causeStore.Get(ctx, causeID)
	if err != nil {
		return fmt.Errorf("failed to get contributing cause: %w", err)
	}

	review.ContributingCauses = append(review.ContributingCauses, ReviewCause{
		Cause: cause,
		Why:   why,
	})

	_, err = s.reviewStore.Save(ctx, review)
	if err != nil {
		return fmt.Errorf("failed to save review: %w", err)
	}

	return nil
}
