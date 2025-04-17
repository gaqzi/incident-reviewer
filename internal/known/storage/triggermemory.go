package storage

import (
	"context"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/known"
)

type TriggerMemoryStore struct {
	data map[uuid.UUID]known.Trigger
}

func NewTriggerMemoryStore() *TriggerMemoryStore {
	return &TriggerMemoryStore{
		data: make(map[uuid.UUID]known.Trigger),
	}
}

func (s *TriggerMemoryStore) Get(_ context.Context, id uuid.UUID) (known.Trigger, error) {
	trigger, ok := s.data[id]
	if !ok {
		return known.Trigger{}, &NoTriggerError{ID: id}
	}
	return trigger, nil
}

func (s *TriggerMemoryStore) Save(_ context.Context, trigger known.Trigger) (known.Trigger, error) {
	if trigger.ID == uuid.Nil {
		return known.Trigger{}, ErrNoID
	}

	s.data[trigger.ID] = trigger

	return trigger, nil
}

func (s *TriggerMemoryStore) All(_ context.Context) ([]known.Trigger, error) {
	ret := make([]known.Trigger, 0, len(s.data))

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
