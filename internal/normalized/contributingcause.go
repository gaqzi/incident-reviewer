package normalized

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/platform/validate"
)

type ContributingCause struct {
	ID          uuid.UUID `validate:"required"`
	Name        string    `validate:"required"`
	Description string    `validate:"required"`
	Category    string    `validate:"required"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewContributingCause() ContributingCause {
	return ContributingCause{ID: uuid.Must(uuid.NewV7())}
}

func (cc ContributingCause) updateTimestamps() ContributingCause {
	now := time.Now()
	if cc.CreatedAt.IsZero() {
		cc.CreatedAt = now
	}
	cc.UpdatedAt = now

	return cc
}

type ContributingCauseService struct {
	store ContributingCauseStorage
}

func NewContributingCauseService(store ContributingCauseStorage) *ContributingCauseService {
	return &ContributingCauseService{store: store}
}

func (s *ContributingCauseService) Save(ctx context.Context, cc ContributingCause) (ContributingCause, error) {
	if err := validate.Struct(ctx, cc); err != nil {
		return cc, fmt.Errorf("failed to validate contributing cause: %w", err)
	}

	cc = cc.updateTimestamps()

	cc, err := s.store.Save(ctx, cc)
	if err != nil {
		return cc, fmt.Errorf("failed to store contributing cause: %w", err)
	}

	return cc, nil
}

func (s *ContributingCauseService) All(ctx context.Context) ([]ContributingCause, error) {
	ret, err := s.store.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get all contributing causes from storage: %w", err)
	}

	return ret, nil
}

func (s *ContributingCauseService) Get(ctx context.Context, id uuid.UUID) (ContributingCause, error) {
	cc, err := s.store.Get(ctx, id)
	if err != nil {
		return ContributingCause{}, fmt.Errorf("failed to get contributing cause: %w", err)
	}

	return cc, nil
}
