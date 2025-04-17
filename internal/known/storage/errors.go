package storage

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type NoCauseError struct {
	ID uuid.UUID
}

func (e *NoCauseError) Error() string {
	return fmt.Sprintf("known cause not found by id: %d", e.ID)
}

// ErrNoID indicates that the passed in uuid ID is blank/uninitialized.
var ErrNoID = errors.New("can't store known cause because ID is not set")

type NoTriggerError struct {
	ID uuid.UUID
}

func (e *NoTriggerError) Error() string {
	return fmt.Sprintf("trigger not found by id: %d", e.ID)
}
