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
