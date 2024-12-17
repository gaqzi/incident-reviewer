package normalized

import "time"

type ContributingCause struct {
	ID          int64
	Name        string `validate:"required"`
	Description string `validate:"required"`
	Category    string `validate:"required"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
