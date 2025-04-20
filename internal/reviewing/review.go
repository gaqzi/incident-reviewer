package reviewing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gaqzi/incident-reviewer/internal/normalized"

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

	BoundCauses   []BoundCause
	BoundTriggers []BoundTrigger

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

// BindContributingCause validates the rc for uniqueness and ensures only one proximal cause at a time.
func (r Review) BindContributingCause(rc BoundCause) (Review, error) {
	// If the new BoundCause is proximal we need to ensure the other ones aren't, so unset when we're iterating over.
	unsetProximal := func(c BoundCause) BoundCause { return c }
	if rc.IsProximalCause {
		unsetProximal = func(c BoundCause) BoundCause {
			c.IsProximalCause = false
			return c
		}
	}

	// Ensure there is always an ID set for the BoundCause
	if rc.ID == uuid.Nil {
		rc.ID = uuid.Must(uuid.NewV7())
	}

	for i, c := range r.BoundCauses {
		if c.Cause.ID == rc.Cause.ID &&
			strings.EqualFold(strings.TrimSpace(c.Why), strings.TrimSpace(rc.Why)) {
			return r, errors.New("cannot bind contributing cause with the same why: " + rc.Why)
		}

		r.BoundCauses[i] = unsetProximal(c)
	}

	r.BoundCauses = append(r.BoundCauses, rc)

	return r, nil
}

func (r Review) UpdateBoundContributingCause(o BoundCause) (Review, error) {
	causes := slices.DeleteFunc(r.BoundCauses, func(rc BoundCause) bool { return rc.ID == o.ID })
	if len(causes) == len(r.BoundCauses) {
		return r, errors.New("cannot update contributing cause that isn't already bound")
	}

	r.BoundCauses = causes
	r, err := r.BindContributingCause(o)
	if err != nil {
		return r, fmt.Errorf("failed to add back bound contributing cause: %w", err)
	}

	return r, nil
}

func (r Review) BindTrigger(t normalized.Trigger, ubt UnboundTrigger) (Review, error) {
	bt := BoundTrigger{
		ID:             uuid.Must(uuid.NewV7()),
		Trigger:        t,
		UnboundTrigger: ubt,
	}

	r.BoundTriggers = append(r.BoundTriggers, bt)

	return r, nil
}

func (r Review) UpdateBoundTrigger(o BoundTrigger) (Review, error) {
	triggers := slices.DeleteFunc(r.BoundTriggers, func(bt BoundTrigger) bool { return bt.ID == o.ID })
	if len(triggers) == len(r.BoundTriggers) {
		return r, errors.New("cannot update trigger that isn't already bound")
	}

	triggers = append(triggers, o)
	r.BoundTriggers = triggers

	return r, nil
}

type BoundCause struct {
	ID              uuid.UUID
	Cause           contributing.Cause `validate:"required"`
	Why             string             `validate:"required"`
	IsProximalCause bool
}

type UnboundTrigger struct {
	Why string `validate:"required"`
}

type BoundTrigger struct {
	ID      uuid.UUID
	Trigger normalized.Trigger `validate:"required"`
	UnboundTrigger
}

func NewBoundCause() BoundCause {
	return BoundCause{ID: uuid.Must(uuid.NewV7())}
}

type causeStore interface {
	Get(ctx context.Context, id uuid.UUID) (contributing.Cause, error)
}

type triggerStore interface {
	Get(ctx context.Context, id uuid.UUID) (normalized.Trigger, error)
}

type Service struct {
	reviewStore  Storage
	causeStore   causeStore
	action       *action.Mapper
	triggerStore triggerStore
}

func (s *Service) BindTrigger(ctx context.Context, reviewID uuid.UUID, triggerID uuid.UUID, unboundTrigger UnboundTrigger) error {
	review, err := s.reviewStore.Get(ctx, reviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}

	trigger, err := s.triggerStore.Get(ctx, triggerID)
	if err != nil {
		return fmt.Errorf("failed to get trigger: %w", err)
	}

	doer, err := s.action.Get("BindTrigger")
	if err != nil {
		return fmt.Errorf("failed to get action for binding trigger: %w", err)
	}
	do, ok := doer.(func(Review, normalized.Trigger, UnboundTrigger) (Review, error))
	if !ok {
		return fmt.Errorf("failed to cast action for binding trigger: %w", err)
	}

	review, err = do(review, trigger, unboundTrigger)
	if err != nil {
		return fmt.Errorf("failed binding trigger to review: %w", err)
	}

	_, err = s.Save(ctx, review)
	if err != nil {
		return fmt.Errorf("failed to save review: %w", err)
	}

	return nil
}

type Option func(s *Service)

func WithActionMapper(mapper *action.Mapper) Option {
	return func(s *Service) {
		s.action = mapper
	}
}

func NewService(reviewStore Storage, causeStore causeStore, triggerStore triggerStore, opts ...Option) *Service {
	s := Service{
		reviewStore:  reviewStore,
		causeStore:   causeStore,
		triggerStore: triggerStore,
		action:       reviewServiceActions(),
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

func (s *Service) BindContributingCause(ctx context.Context, reviewID uuid.UUID, causeID uuid.UUID, boundCause BoundCause) error {
	review, err := s.reviewStore.Get(ctx, reviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}

	cause, err := s.causeStore.Get(ctx, causeID)
	if err != nil {
		return fmt.Errorf("failed to get contributing cause: %w", err)
	}

	doer, err := s.action.Get("BindContributingCause")
	if err != nil {
		return fmt.Errorf("failed to get action for adding contributing cause: %w", err)
	}
	do, ok := doer.(func(Review, contributing.Cause, BoundCause) (Review, error))
	if !ok {
		return fmt.Errorf("failed to cast action for adding contributing cause: %w", err)
	}

	review, err = do(review, cause, boundCause)
	if err != nil {
		return fmt.Errorf("failed to add contributing cause to review: %w", err)
	}

	_, err = s.Save(ctx, review)
	if err != nil {
		return fmt.Errorf("failed to save review: %w", err)
	}

	return nil
}

func (s *Service) GetBoundContributingCause(ctx context.Context, reviewID uuid.UUID, boundCauseID uuid.UUID) (BoundCause, error) {
	review, err := s.reviewStore.Get(ctx, reviewID)
	if err != nil {
		return BoundCause{}, fmt.Errorf("review with that id not found to relate bound contributing cause: %w", err)
	}

	// Check for the bound contributing cause within the review
	for _, boundCause := range review.BoundCauses {
		if boundCause.ID == boundCauseID {
			return boundCause, nil
		}
	}

	return BoundCause{}, errors.New("review doesn't have that contributing cause bound: " + boundCauseID.String())
}

func (s *Service) UpdateBoundContributingCause(ctx context.Context, reviewID uuid.UUID, update BoundCause) (BoundCause, error) {
	review, err := s.reviewStore.Get(ctx, reviewID)
	if err != nil {
		return BoundCause{}, fmt.Errorf("failed to get review: %w", err)
	}

	newCause, err := s.causeStore.Get(ctx, update.Cause.ID)
	if err != nil {
		return BoundCause{}, fmt.Errorf("failed to get contributing cause: %w", err)
	}
	update.Cause = newCause

	doer, err := s.action.Get("UpdateBoundContributingCause")
	if err != nil {
		return BoundCause{}, fmt.Errorf("failed to get action for updating bound contributing cause: %w", err)
	}
	do, ok := doer.(func(Review, BoundCause) (Review, error))
	if !ok {
		return BoundCause{}, fmt.Errorf("failed to cast action for updating bound contributing cause: %w", err)
	}

	review, err = do(review, update)
	if err != nil {
		return BoundCause{}, fmt.Errorf("action to update bound contributing cause failed: %w", err)
	}

	updatedReview, err := s.Save(ctx, review)
	if err != nil {
		return BoundCause{}, fmt.Errorf("failed to save updated review: %w", err)
	}

	// Return the updated contributing cause
	for _, boundCause := range updatedReview.BoundCauses {
		if boundCause.ID == update.ID {
			return boundCause, nil
		}
	}

	return BoundCause{}, errors.New("unexpected error: updated contributing cause not found")
}

func (s *Service) GetBoundTrigger(ctx context.Context, reviewID uuid.UUID, boundTriggerID uuid.UUID) (BoundTrigger, error) {
	review, err := s.reviewStore.Get(ctx, reviewID)
	if err != nil {
		return BoundTrigger{}, fmt.Errorf("review with that id not found to relate bound trigger: %w", err)
	}

	// Check for the bound trigger within the review
	for _, boundTrigger := range review.BoundTriggers {
		if boundTrigger.ID == boundTriggerID {
			return boundTrigger, nil
		}
	}

	return BoundTrigger{}, errors.New("review doesn't have that trigger bound: " + boundTriggerID.String())
}

func (s *Service) UpdateBoundTrigger(ctx context.Context, reviewID uuid.UUID, update BoundTrigger) (BoundTrigger, error) {
	review, err := s.reviewStore.Get(ctx, reviewID)
	if err != nil {
		return BoundTrigger{}, fmt.Errorf("failed to get review: %w", err)
	}

	newTrigger, err := s.triggerStore.Get(ctx, update.Trigger.ID)
	if err != nil {
		return BoundTrigger{}, fmt.Errorf("failed to get trigger: %w", err)
	}
	update.Trigger = newTrigger

	doer, err := s.action.Get("UpdateBoundTrigger")
	if err != nil {
		return BoundTrigger{}, fmt.Errorf("failed to get action for updating bound trigger: %w", err)
	}
	do, ok := doer.(func(Review, BoundTrigger) (Review, error))
	if !ok {
		return BoundTrigger{}, fmt.Errorf("failed to cast action for updating bound trigger: %w", err)
	}

	review, err = do(review, update)
	if err != nil {
		return BoundTrigger{}, fmt.Errorf("action to update bound trigger failed: %w", err)
	}

	updatedReview, err := s.Save(ctx, review)
	if err != nil {
		return BoundTrigger{}, fmt.Errorf("failed to save updated review: %w", err)
	}

	// Return the updated trigger
	for _, boundTrigger := range updatedReview.BoundTriggers {
		if boundTrigger.ID == update.ID {
			return boundTrigger, nil
		}
	}

	return BoundTrigger{}, errors.New("unexpected error: updated trigger not found")
}
