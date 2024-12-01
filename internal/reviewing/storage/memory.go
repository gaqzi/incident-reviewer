package storage

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/gaqzi/incident-reviewer/internal/reviewing"
)

type MemoryStore struct {
	data      map[int64]reviewing.Review
	currentID int64
	validate  *validator.Validate
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data:     make(map[int64]reviewing.Review),
		validate: validator.New(),
	}
}

func (s *MemoryStore) Save(_ context.Context, inc reviewing.Review) (reviewing.Review, error) {
	if err := s.validate.Struct(inc); err != nil {
		return reviewing.Review{}, fmt.Errorf("failed to validate incident: %w", err)
	}

	if inc.ID == 0 {
		s.currentID++
		inc.ID = s.currentID
	}

	s.data[inc.ID] = inc

	return inc, nil
}

func (s *MemoryStore) Get(_ context.Context, ID int64) (reviewing.Review, error) {
	review, ok := s.data[ID]
	if !ok {
		return reviewing.Review{}, &NoReviewError{ID: ID}
	}

	return review, nil
}
