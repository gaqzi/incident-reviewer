package storage

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
)

// TODO: refactor into a generic implementation because the logic is the same across this one and reviewing/storage.MemoryStore.

type ContributingCauseMemoryStore struct {
	data      map[int64]normalized.ContributingCause
	currentID int64
	validate  *validator.Validate
}

func NewContributingCauseMemoryStore() *ContributingCauseMemoryStore {
	return &ContributingCauseMemoryStore{
		data:     make(map[int64]normalized.ContributingCause),
		validate: validator.New(),
	}
}

func (s *ContributingCauseMemoryStore) Get(_ context.Context, id int64) (normalized.ContributingCause, error) {
	cause, ok := s.data[id]
	if !ok {
		return normalized.ContributingCause{}, &NoContributingCauseError{ID: id}
	}

	return cause, nil
}

func (s *ContributingCauseMemoryStore) Save(_ context.Context, cause normalized.ContributingCause) (normalized.ContributingCause, error) {
	if err := s.validate.Struct(cause); err != nil {
		return normalized.ContributingCause{}, fmt.Errorf("failed to validate cause: %w", err)
	}

	if cause.ID == 0 {
		s.currentID++
		cause.ID = s.currentID
	}

	now := time.Now()
	if cause.CreatedAt.IsZero() {
		cause.CreatedAt = now
	}
	cause.UpdatedAt = now

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
