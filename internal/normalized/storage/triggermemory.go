package storage

import (
	"context"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing/storage"
)

type TriggerMemoryStore struct {
	data map[uuid.UUID]normalized.Trigger
}

func NewTriggerMemoryStore() *TriggerMemoryStore {
	return &TriggerMemoryStore{
		data: make(map[uuid.UUID]normalized.Trigger),
	}
}

func (s *TriggerMemoryStore) Get(_ context.Context, id uuid.UUID) (normalized.Trigger, error) {
	trigger, ok := s.data[id]
	if !ok {
		return normalized.Trigger{}, &storage.NoTriggerError{ID: id}
	}
	return trigger, nil
}

func (s *TriggerMemoryStore) Save(_ context.Context, trigger normalized.Trigger) (normalized.Trigger, error) {
	if trigger.ID == uuid.Nil {
		return normalized.Trigger{}, storage.ErrNoID
	}

	s.data[trigger.ID] = trigger

	return trigger, nil
}

func (s *TriggerMemoryStore) All(_ context.Context) ([]normalized.Trigger, error) {
	ret := make([]normalized.Trigger, 0, len(s.data))

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
