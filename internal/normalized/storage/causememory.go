package storage

import (
	"context"
	"maps"
	"slices"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
)

// TODO: refactor into a generic implementation because the logic is the same across this one and reviewing/storage.MemoryStore.

type ContributingCauseMemoryStore struct {
	data      map[int64]normalized.ContributingCause
	currentID int64
}

func NewContributingCauseMemoryStore() *ContributingCauseMemoryStore {
	return &ContributingCauseMemoryStore{
		data: make(map[int64]normalized.ContributingCause),
	}
}

func (s *ContributingCauseMemoryStore) Get(_ context.Context, id int64) (normalized.ContributingCause, error) {
	cause, ok := s.data[id]
	if !ok {
		return normalized.ContributingCause{}, &NoContributingCauseError{ID: id}
	}

	return cause, nil
}

func (s *ContributingCauseMemoryStore) Save(ctx context.Context, cause normalized.ContributingCause) (normalized.ContributingCause, error) {
	if cause.ID == 0 {
		s.currentID++
		cause.ID = s.currentID
	}

	s.data[cause.ID] = cause

	return cause, nil
}

func (s *ContributingCauseMemoryStore) All(_ context.Context) ([]normalized.ContributingCause, error) {
	ret := make([]normalized.ContributingCause, 0, len(s.data))

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
