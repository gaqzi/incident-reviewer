package storage

import "fmt"

type NoContributingCauseError struct {
	ID int64
}

func (e *NoContributingCauseError) Error() string {
	return fmt.Sprintf("contributing cause not found by id: %d", e.ID)
}
