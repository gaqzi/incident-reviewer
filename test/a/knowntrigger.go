package a

import (
	"time"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/known"
)

type BuilderKnownTrigger struct {
	t known.Trigger
}

func (b BuilderKnownTrigger) IsValid() BuilderKnownTrigger {
	b.t.ID = uuid.MustParse("0193ddee-c2e6-72d6-ad36-9d4cee8a5e2f") // UUIDv7, just a value, no particular meaning
	b.t.Name = "Third Party Outage"
	b.t.Description = "When things go wrong for others"

	return b
}

func (b BuilderKnownTrigger) IsSaved() BuilderKnownTrigger {
	createdAt, err := time.Parse(time.RFC3339Nano, "2025-03-06T07:25:30.1337Z")
	if err != nil {
		panic("failed to parse example timestamp: " + err.Error())
	}

	b.t.CreatedAt = createdAt
	b.t.UpdatedAt = createdAt

	return b
}

func (b BuilderKnownTrigger) Build() known.Trigger {
	return b.t
}

func (b BuilderKnownTrigger) IsNotSaved() BuilderKnownTrigger {
	b.t.CreatedAt = time.Time{}
	b.t.UpdatedAt = time.Time{}

	return b
}

func (b BuilderKnownTrigger) WithID(uuid uuid.UUID) BuilderKnownTrigger {
	b.t.ID = uuid
	return b
}

func (b BuilderKnownTrigger) WithName(n string) BuilderKnownTrigger {
	b.t.Name = n
	return b
}

func Trigger() BuilderKnownTrigger {
	return BuilderKnownTrigger{}.
		IsValid().
		IsSaved()
}
