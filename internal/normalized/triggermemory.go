package normalized

import (
	"context"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing/storage"
)

type TriggerMemoryStore struct {
	data map[uuid.UUID]Trigger
}

func NewTriggerMemoryStore() *TriggerMemoryStore {
	return &TriggerMemoryStore{
		data: make(map[uuid.UUID]Trigger),
	}
}

func (s *TriggerMemoryStore) Get(_ context.Context, id uuid.UUID) (Trigger, error) {
	trigger, ok := s.data[id]
	if !ok {
		return Trigger{}, &storage.NoTriggerError{ID: id}
	}
	return trigger, nil
}

func (s *TriggerMemoryStore) Save(_ context.Context, trigger Trigger) (Trigger, error) {
	if trigger.ID == uuid.Nil {
		return Trigger{}, storage.NoIDError
	}

	s.data[trigger.ID] = trigger

	return trigger, nil
}
