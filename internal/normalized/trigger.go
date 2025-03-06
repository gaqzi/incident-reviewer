package normalized

import (
	"github.com/google/uuid"
	"time"
)

type Trigger struct {
	ID          uuid.UUID `validate:"required"`
	Name        string    `validate:"required"`
	Description string    `validate:"required"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
