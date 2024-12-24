package a

import "github.com/google/uuid"

// UUID returns a new random UUID.
func UUID() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}
