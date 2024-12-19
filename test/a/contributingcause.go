package a

import (
	"time"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
)

type BuilderContributingCause struct {
	c normalized.ContributingCause
}

func ContributingCause() BuilderContributingCause {
	return BuilderContributingCause{}.
		IsValid().
		IsSaved()
}

func (b BuilderContributingCause) IsValid() BuilderContributingCause {
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

	b.c.ID = 1
	b.c.CreatedAt = createdAt
	b.c.UpdatedAt = createdAt

	return b
}

func (b BuilderContributingCause) IsNotSaved() BuilderContributingCause {
	b.c.ID = 0
	b.c.CreatedAt = time.Time{}
	b.c.UpdatedAt = time.Time{}

	return b
}

func (b BuilderContributingCause) WithID(id int64) BuilderContributingCause {
	b.c.ID = id

	return b
}

func (b BuilderContributingCause) WithName(n string) BuilderContributingCause {
	b.c.Name = n

	return b
}

func (b BuilderContributingCause) Modify(mods ...func(cc *normalized.ContributingCause)) BuilderContributingCause {
	for _, m := range mods {
		m(&b.c)
	}

	return b

}

func (b BuilderContributingCause) Build() normalized.ContributingCause {
	return b.c
}
