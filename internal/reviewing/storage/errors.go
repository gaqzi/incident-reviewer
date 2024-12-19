package storage

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type NoReviewError struct {
	ID uuid.UUID
}

func (e *NoReviewError) Error() string {
	return fmt.Sprintf("review not found by id: %d", e.ID)
}

// NoIDError indicates that the passed in uuid ID is blank/uninitialized.
var NoIDError = errors.New("can't store review because ID is not set")
