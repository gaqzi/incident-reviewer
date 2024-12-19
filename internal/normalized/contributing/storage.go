package contributing

import (
	"context"

	"github.com/google/uuid"
)

type CauseStorage interface {
	Get(ctx context.Context, id uuid.UUID) (Cause, error)

	Save(ctx context.Context, cause Cause) (Cause, error)

	All(ctx context.Context) ([]Cause, error)
}
