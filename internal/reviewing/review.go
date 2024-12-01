package reviewing

import "time"

type Review struct {
	ID          int64
	URL         string `validate:"required,http_url"`
	Title       string `validate:"required"`
	Description string `validate:"required"`
	Impact      string `validate:"required"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
