package a

import (
	"time"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
)

type BuilderContributingCause struct {
	c contributing.Cause
}

func ContributingCause() BuilderContributingCause {
	return BuilderContributingCause{}.
		IsValid().
		IsSaved()
}

func (b BuilderContributingCause) IsValid() BuilderContributingCause {
	b.c.ID = uuid.MustParse("0193ddee-c2e6-72d6-ad36-9d4cee8a5e2f") // UUIDv7, just a value, no particular meaning
	b.c.Name = "Third Party Outage"
	b.c.Description = "When things go wrong for others"
	b.c.Category = "Design" // because we can mitigate these by designing differently, mostly

	return b
}

func (b BuilderContributingCause) IsInvalid() BuilderContributingCause {
	b.c.Name = ""
	b.c.Description = ""
	b.c.Category = ""

	return b
}

func (b BuilderContributingCause) IsSaved() BuilderContributingCause {
	createdAt, err := time.Parse(time.RFC3339Nano, "2024-12-19T07:25:30.1337Z")
	if err != nil {
		panic("failed to parse example timestamp: " + err.Error())
	}

	b.c.CreatedAt = createdAt
	b.c.UpdatedAt = createdAt

	return b
}

func (b BuilderContributingCause) IsNotSaved() BuilderContributingCause {
	b.c.CreatedAt = time.Time{}
	b.c.UpdatedAt = time.Time{}

	return b
}

func (b BuilderContributingCause) WithID(id uuid.UUID) BuilderContributingCause {
	b.c.ID = id

	return b
}

func (b BuilderContributingCause) WithName(n string) BuilderContributingCause {
	b.c.Name = n

	return b
}

func (b BuilderContributingCause) Modify(mods ...func(cc *contributing.Cause)) BuilderContributingCause {
	for _, m := range mods {
		m(&b.c)
	}

	return b

}

func (b BuilderContributingCause) Build() contributing.Cause {
	return b.c
}
