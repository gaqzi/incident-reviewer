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

// ErrNoID indicates that the passed in ID is blank/uninitialized.
var ErrNoID = errors.New("can't store review because ID is not set")
