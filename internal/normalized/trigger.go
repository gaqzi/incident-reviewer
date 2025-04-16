package normalized

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/platform/validate"
)

type Trigger struct {
	ID          uuid.UUID `validate:"required"`
	Name        string    `validate:"required"`
	Description string    `validate:"required"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (t Trigger) updateTimestamps() Trigger {
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	return t
}

func NewTrigger() Trigger {
	return Trigger{ID: uuid.Must(uuid.NewV7())}
}

type TriggerService struct {
	store TriggerStorage
}

func NewTriggerService(store TriggerStorage) *TriggerService {
	return &TriggerService{store}
}

func (s *TriggerService) Save(ctx context.Context, t Trigger) (Trigger, error) {
	if err := validate.Struct(ctx, t); err != nil {
		return t, fmt.Errorf("failed to validate trigger: %w", err)
	}

	t = t.updateTimestamps()

	t, err := s.store.Save(ctx, t)
	if err != nil {
		return t, fmt.Errorf("failed to store normalized trigger: %w", err)
	}

	return t, nil
}

func (s *TriggerService) All(ctx context.Context) ([]Trigger, error) {
	ret, err := s.store.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get all triggers from storage: %w", err)
	}

	return ret, nil
}

func (s *TriggerService) Get(ctx context.Context, id uuid.UUID) (Trigger, error) {
	t, err := s.store.Get(ctx, id)
	if err != nil {
		return Trigger{}, fmt.Errorf("failed to get trigger: %w", err)
	}

	return t, nil
}
