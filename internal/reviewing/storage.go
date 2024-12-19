package reviewing

import (
	"context"

	"github.com/google/uuid"
)

type Storage interface {
	// Save saves the review or if it fails validation return an error with all failures.
	Save(ctx context.Context, review Review) (Review, error)

	// Get finds the review or returns NotFoundError.
	Get(ctx context.Context, ID uuid.UUID) (Review, error)

	// All returns all the stored reviews with the most recent first.
	All(ctx context.Context) ([]Review, error)
}
