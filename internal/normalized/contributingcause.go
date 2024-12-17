package normalized

import (
	"context"
	"fmt"
	"time"
)

type ContributingCause struct {
	ID          int64
	Name        string `validate:"required"`
	Description string `validate:"required"`
	Category    string `validate:"required"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ContributingCauseService struct {
	store ContributingCauseStorage
}

func NewContributingCauseService(store ContributingCauseStorage) *ContributingCauseService {
	return &ContributingCauseService{store: store}
}

func (s *ContributingCauseService) All(ctx context.Context) ([]ContributingCause, error) {
	ret, err := s.store.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get all contributing causes from storage: %w", err)
	}

	return ret, nil
}
