package normalized

import (
	"time"

	"github.com/google/uuid"
)

type Trigger struct {
	ID          uuid.UUID `validate:"required"`
	Name        string    `validate:"required"`
	Description string    `validate:"required"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
