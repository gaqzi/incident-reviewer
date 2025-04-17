package known

import (
	"context"

	"github.com/google/uuid"
)

type TriggerStorage interface {
	Get(ctx context.Context, id uuid.UUID) (Trigger, error)

	Save(ctx context.Context, trigger Trigger) (Trigger, error)

	All(ctx context.Context) ([]Trigger, error)
}

type CauseStorage interface {
	Get(ctx context.Context, id uuid.UUID) (Cause, error)

	Save(ctx context.Context, cause Cause) (Cause, error)

	All(ctx context.Context) ([]Cause, error)
}
