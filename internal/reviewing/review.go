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

type Service struct {
	reviewStore Storage
	causeStore  normalized.ContributingCauseStorage
}

func NewService(reviewStore Storage, causeStore normalized.ContributingCauseStorage) *Service {
	return &Service{
		reviewStore: reviewStore,
		causeStore:  causeStore,
	}
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
