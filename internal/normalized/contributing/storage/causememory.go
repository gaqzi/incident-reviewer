package storage

import (
	"context"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
)

// TODO: refactor into a generic implementation because the logic is the same across this one and reviewing/storage.MemoryStore.

type CauseMemoryStore struct {
	data map[uuid.UUID]contributing.Cause
}

func NewCauseMemoryStore() *CauseMemoryStore {
	return &CauseMemoryStore{
		data: make(map[uuid.UUID]contributing.Cause),
	}
}

func (s *CauseMemoryStore) Get(_ context.Context, id uuid.UUID) (contributing.Cause, error) {
	cause, ok := s.data[id]
	if !ok {
		return contributing.Cause{}, &NoCauseError{ID: id}
	}

	return cause, nil
}

func (s *CauseMemoryStore) Save(_ context.Context, cause contributing.Cause) (contributing.Cause, error) {
	if cause.ID == uuid.Nil {
		return contributing.Cause{}, ErrNoID
	}

	s.data[cause.ID] = cause

	return cause, nil
}

func (s *CauseMemoryStore) All(_ context.Context) ([]contributing.Cause, error) {
	ret := make([]contributing.Cause, 0, len(s.data))

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
