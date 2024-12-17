package storage

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/gaqzi/incident-reviewer/internal/platform/validate"
	"github.com/gaqzi/incident-reviewer/internal/reviewing"
)

type MemoryStore struct {
	data      map[int64]reviewing.Review
	currentID int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[int64]reviewing.Review),
	}
}

func (s *MemoryStore) Save(ctx context.Context, inc reviewing.Review) (reviewing.Review, error) {
	if err := validate.Struct(ctx, inc); err != nil {
		return reviewing.Review{}, fmt.Errorf("failed to validate incident: %w", err)
	}

	if inc.ID == 0 {
		s.currentID++
		inc.ID = s.currentID
	}

	now := time.Now()
	if inc.CreatedAt.IsZero() {
		inc.CreatedAt = now
	}
	inc.UpdatedAt = now

	s.data[inc.ID] = inc

	return inc, nil
}

func (s *MemoryStore) Get(_ context.Context, id int64) (reviewing.Review, error) {
	review, ok := s.data[id]
	if !ok {
		return reviewing.Review{}, &NoReviewError{ID: id}
	}

	return review, nil
}

func (s *MemoryStore) All(_ context.Context) ([]reviewing.Review, error) {
	ret := make([]reviewing.Review, 0, len(s.data))

	// Sort all the keys for the store, which returns keys in a non-deterministic order,
	// and the sort order is 1, 2, 3… by the ID, which is monotonically incrementing
	keys := slices.Sorted(maps.Keys(s.data))
	// and since I want them returned with most recent first, reverse it after sorting
	slices.Reverse(keys)

	for _, r := range keys {
		ret = append(ret, s.data[r])
	}

	return ret, nil
}
