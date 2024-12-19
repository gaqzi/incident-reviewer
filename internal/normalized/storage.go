package normalized

import (
	"context"

	"github.com/google/uuid"
)

type ContributingCauseStorage interface {
	Get(ctx context.Context, id uuid.UUID) (ContributingCause, error)

	Save(ctx context.Context, cause ContributingCause) (ContributingCause, error)

	All(ctx context.Context) ([]ContributingCause, error)
}
