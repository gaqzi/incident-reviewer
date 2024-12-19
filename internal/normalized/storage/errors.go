package storage

import (
	"errors"
	"fmt"
)

type NoContributingCauseError struct {
	ID int64
}

func (e *NoContributingCauseError) Error() string {
	return fmt.Sprintf("contributing cause not found by id: %d", e.ID)
}

// NoIDError indicates that the passed in uuid ID is blank/uninitialized.
var NoIDError = errors.New("can't store contributing cause because ID is not set")
