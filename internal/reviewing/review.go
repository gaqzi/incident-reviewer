package reviewing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
	"github.com/gaqzi/incident-reviewer/internal/platform/action"
)

type Review struct {
	ID                  uuid.UUID `validate:"required"`
	URL                 string    `validate:"required,http_url"`
	Title               string    `validate:"required"`
	Description         string    `validate:"required"`
	Impact              string    `validate:"required"`
	Where               string    `validate:"required"`
	ReportProximalCause string    `validate:"required"`
	ReportTrigger       string    `validate:"required"`

	ContributingCauses []ReviewCause

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewReview returns a reviewing.Review with a valid ID set.
func NewReview() Review {
	return Review{ID: uuid.Must(uuid.NewV7())}
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

// updateTimestamps is intended to be used before storing the Review to make tracking changes easier.
// It's kept private because it'll be called by the service, and I'm curious about this design decision,
// but it seems like the best way of making it exist while also keeping the service not involved in the logic.
func (r Review) updateTimestamps() Review {
	now := time.Now()

	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now

	return r
}

// AddContributingCause validates the rc for uniqueness and ensures only one proximal cause at a time.
func (r Review) AddContributingCause(rc ReviewCause) (Review, error) {
	// If the new ReviewCause is proximal we need to ensure the other ones aren't, so unset when we're iterating over.
	unsetProximal := func(c ReviewCause) ReviewCause { return c }
	if rc.IsProximalCause {
		unsetProximal = func(c ReviewCause) ReviewCause {
			c.IsProximalCause = false
			return c
		}
	}

	for i, c := range r.ContributingCauses {
		if c.Cause.ID == rc.Cause.ID &&
			strings.EqualFold(strings.TrimSpace(c.Why), strings.TrimSpace(rc.Why)) {
			return r, errors.New("cannot bind contributing cause with the same why: " + rc.Why)
		}

		r.ContributingCauses[i] = unsetProximal(c)
	}

	r.ContributingCauses = append(r.ContributingCauses, rc)

	return r, nil
}

type ReviewCause struct {
	Cause           contributing.Cause `validate:"required"`
	Why             string             `validate:"required"`
	IsProximalCause bool
}

type causeStore interface {
	Get(ctx context.Context, id uuid.UUID) (contributing.Cause, error)
}

type Service struct {
	reviewStore Storage
	causeStore  causeStore
	action      *action.Mapper
}

type Option func(s *Service)

func WithActionMapper(mapper *action.Mapper) Option {
	return func(s *Service) {
		s.action = mapper
	}
}

func NewService(reviewStore Storage, causeStore causeStore, opts ...Option) *Service {
	s := Service{
		reviewStore: reviewStore,
		causeStore:  causeStore,
		action:      reviewServiceActions(),
	}

	for _, opt := range opts {
		opt(&s)
	}

	return &s
}

func (s *Service) Save(ctx context.Context, review Review) (Review, error) {
	doer, err := s.action.Get("Save")
	if err != nil {
		return review, fmt.Errorf("failed to get action for save: %w", err)
	}
	do, ok := doer.(func(context.Context, Review) (Review, error))
	if !ok {
		return review, errors.New("action for save doesn't match")
	}

	review, err = do(ctx, review)
	if err != nil {
		return review, fmt.Errorf("pre-save action failed: %w", err)
	}

	review, err = s.reviewStore.Save(ctx, review)
	if err != nil {
		return Review{}, fmt.Errorf("failed to save review in storage: %w", err)
	}

	return review, nil
}

func (s *Service) Get(ctx context.Context, reviewID uuid.UUID) (Review, error) {
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

func (s *Service) AddContributingCause(ctx context.Context, reviewID uuid.UUID, causeID uuid.UUID, reviewCause ReviewCause) error {
	review, err := s.reviewStore.Get(ctx, reviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}

	cause, err := s.causeStore.Get(ctx, causeID)
	if err != nil {
		return fmt.Errorf("failed to get contributing cause: %w", err)
	}

	doer, err := s.action.Get("AddContributingCause")
	if err != nil {
		return fmt.Errorf("failed to get action for adding contributing cause: %w", err)
	}
	do, ok := doer.(func(Review, contributing.Cause, ReviewCause) (Review, error))
	if !ok {
		return fmt.Errorf("failed to cast action for adding contributing cause: %w", err)
	}

	review, err = do(review, cause, reviewCause)
	if err != nil {
		return fmt.Errorf("failed to add contributing cause to review: %w", err)
	}

	_, err = s.Save(ctx, review)
	if err != nil {
		return fmt.Errorf("failed to save review: %w", err)
	}

	return nil
}
