package action

import (
	"errors"
	"maps"
	"slices"
)

// Mapper provides "dynamic" injection of actions in services.
// The idea is to make sure your service objects keep doing collaboration while also
// giving them a way to declare how they want to collaborate with their aggregate roots,
// which perform business logic, and may need to be called to let the service object
// perform their various actions.
// So Instead of forcing each caller to keep track of how to perform all the steps of the
// action, we can name it, provide a default, and use that during normal execution, and
// then provide a mocked version in testing.
type Mapper struct {
	actions map[string]any
}

func (m *Mapper) Add(name string, fn any) *Mapper {
	if m.actions == nil {
		m.actions = make(map[string]any)
	}

	m.actions[name] = fn

	return m
}

func (m *Mapper) Get(name string) (any, error) {
	v, ok := m.actions[name]
	if !ok {
		return nil, errors.New("no action found for: " + name)
	}

	return v, nil
}

func (m *Mapper) All() []string {
	return slices.Collect(maps.Keys(m.actions))
}
