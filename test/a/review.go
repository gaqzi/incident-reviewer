// Package a happily stolen from Working Effectively with Unit Tests.
package a

import (
	"time"

	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/reviewing"
)

type BuilderReview struct {
	r reviewing.Review
}

// Review prepares a reviewing.Review that is valid and saved by default but allows for customization.
func Review() BuilderReview {
	r := BuilderReview{}

	return r.IsValid().IsSaved()
}

// Build returns the prepared reviewing.Review.
func (b BuilderReview) Build() reviewing.Review {
	return b.r
}

// IsInvalid prepares a reviewing.Review that will fail validation.
func (b BuilderReview) IsInvalid() BuilderReview {
	b.r = reviewing.Review{}

	return b
}

// IsValid prepares a reviewing.Review that will pass validation.
func (b BuilderReview) IsValid() BuilderReview {
	b.r.ID = uuid.MustParse("0193dd86-b07e-7e73-a77e-724bee1fa176") // UUIDv7, just a value, no particular meaning
	b.r.URL = "https://example.com/reviews/1"
	b.r.Title = "Something"
	b.r.Description = "At the bottom of the sea"
	b.r.Impact = "did a bunch of things"
	b.r.Where = "At land"
	b.r.ReportProximalCause = "Broken"
	b.r.ReportTrigger = "Special operation"

	return b
}

// IsSaved prepares a reviewing.Review that has previously been saved.
// This means we've provided valid:
// - CreatedAt
// - UpdatedAt
func (b BuilderReview) IsSaved() BuilderReview {
	createdAt, err := time.Parse(time.RFC3339Nano, "2024-12-17T18:50:02.1323Z")
	if err != nil {
		panic("failed to parse example timestamp: " + err.Error())
	}
	b.r.CreatedAt = createdAt
	b.r.UpdatedAt = createdAt

	return b
}

// IsNotSaved prepares a reviewing.Review that has not been saved.
func (b BuilderReview) IsNotSaved() BuilderReview {
	b.r.CreatedAt = time.Time{}
	b.r.UpdatedAt = time.Time{}

	return b
}

// WithID prepares the reviewing.Review with the passed in id.
func (b BuilderReview) WithID(id uuid.UUID) BuilderReview {
	b.r.ID = id

	return b
}

func (b BuilderReview) WithURL(url string) BuilderReview {
	b.r.URL = url

	return b
}

// Modify allows you to specify a custom override while preparing.
// Note: consider naming your pattern and adding it to the builder.
func (b BuilderReview) Modify(mods ...func(r *reviewing.Review)) BuilderReview {
	for _, mod := range mods {
		mod(&b.r)
	}

	return b
}

func (b BuilderReview) WithContributingCause(rc reviewing.ReviewCause) BuilderReview {
	r, err := b.r.AddContributingCause(rc)
	if err != nil {
		panic("failed to add contributing cause: " + err.Error())
	}
	b.r = r

	return b
}
