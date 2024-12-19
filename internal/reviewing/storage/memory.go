package storage

import (
	"context"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/reviewing"
)

type MemoryStore struct {
	data map[uuid.UUID]reviewing.Review
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[uuid.UUID]reviewing.Review),
	}
}

func (s *MemoryStore) Save(_ context.Context, inc reviewing.Review) (reviewing.Review, error) {
	if inc.ID == uuid.Nil {
		return reviewing.Review{}, NoIDError
	}

	s.data[inc.ID] = inc

	return inc, nil
}

func (s *MemoryStore) Get(_ context.Context, id uuid.UUID) (reviewing.Review, error) {
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
	keys := slices.SortedFunc(maps.Keys(s.data), func(u uuid.UUID, u2 uuid.UUID) int {
		t1, t2 := u.Time(), u2.Time()
		switch {
		case t1 < t2:
			return -1
		case t1 > t2:
			return 1
		default:
			return 0
		}
	})
	// and since I want them returned with most recent first, reverse it after sorting
	slices.Reverse(keys)

	for _, r := range keys {
		ret = append(ret, s.data[r])
	}

	return ret, nil
}
