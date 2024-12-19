package contributing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/platform/validate"
)

type Cause struct {
	ID          uuid.UUID `validate:"required"`
	Name        string    `validate:"required"`
	Description string    `validate:"required"`
	Category    string    `validate:"required"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCause() Cause {
	return Cause{ID: uuid.Must(uuid.NewV7())}
}

func (cc Cause) updateTimestamps() Cause {
	now := time.Now()
	if cc.CreatedAt.IsZero() {
		cc.CreatedAt = now
	}
	cc.UpdatedAt = now

	return cc
}

type CauseService struct {
	store CauseStorage
}

func NewCauseService(store CauseStorage) *CauseService {
	return &CauseService{store: store}
}

func (s *CauseService) Save(ctx context.Context, cc Cause) (Cause, error) {
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

func (s *CauseService) All(ctx context.Context) ([]Cause, error) {
	ret, err := s.store.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get all contributing causes from storage: %w", err)
	}

	return ret, nil
}

func (s *CauseService) Get(ctx context.Context, id uuid.UUID) (Cause, error) {
	cc, err := s.store.Get(ctx, id)
	if err != nil {
		return Cause{}, fmt.Errorf("failed to get contributing cause: %w", err)
	}

	return cc, nil
}
