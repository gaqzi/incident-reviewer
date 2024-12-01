package storage

import "fmt"

type NoReviewError struct {
	ID int64
}

func (e *NoReviewError) Error() string {
	return fmt.Sprintf("review not found by id: %d", e.ID)
}
