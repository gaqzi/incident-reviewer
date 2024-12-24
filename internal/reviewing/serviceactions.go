package reviewing

import (
	"context"
	"fmt"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
	"github.com/gaqzi/incident-reviewer/internal/platform/action"
	"github.com/gaqzi/incident-reviewer/internal/platform/validate"
)

// reviewServiceActions provides actions to be taken as pre-hooks in my reviewing.Service.
// It's a way for me to keep certain logic private (so it doesn't leak in my public interface) while
// also keeping the service clean in that it only does collaboration.
// I'm not 100% about this pattern, but I'm making pieces that are either about collaboration or logic, so it's
// something.
func reviewServiceActions() *action.Mapper {
	m := &action.Mapper{}

	m.Add("AddContributingCause", func(r Review, c contributing.Cause, rc ReviewCause) (Review, error) {
		rc.Cause = c
		return r.AddContributingCause(rc)
	})

	m.Add("UpdateBoundContributingCause", func(r Review, o ReviewCause) (Review, error) {
		return r.UpdateBoundContributingCause(o)
	})

	m.Add("Save", func(ctx context.Context, r Review) (Review, error) {
		if err := validate.Struct(ctx, r); err != nil {
			return r, fmt.Errorf("failed to validate review: %w", err)
		}

		return r.updateTimestamps(), nil
	})

	return m
}
