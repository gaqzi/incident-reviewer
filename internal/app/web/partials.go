package web

import (
	"github.com/donseba/go-htmx"
	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
)

type htmxComponentAttacher func(c htmx.RenderableComponent) htmx.RenderableComponent

// attachToComponent acts as a pipeline wrapper to a component, to make it so you can see that all the functions are applied one after another.
func attachToComponent(component htmx.RenderableComponent, attachers ...htmxComponentAttacher) htmx.RenderableComponent {
	for _, attach := range attachers {
		attach(component)
	}

	return component
}

func bindContributingCausesOptions(ccs []contributing.Cause) htmxComponentAttacher {
	return func(c htmx.RenderableComponent) htmx.RenderableComponent {
		return c.
			Attach("templates/contributing-causes/binding/__causes-options.html").
			AddData("ContributingCauses", convertContributingCauseToHttpObjects(ccs))
	}
}

// contributingCausesComponent constructs the <contributing-causes> template for binding and listing contributing causes to a review.
// TODO: replace []contributing.Cause with []ContributingCauseBasic.
func contributingCausesComponent(reviewID uuid.UUID, ccs []contributing.Cause, boundCauses []ReviewCauseBasic) htmx.RenderableComponent {
	return htmx.
		NewComponent("templates/reviews/_contributing-causes.html").
		FS(templates).
		With(
			attachToComponent(
				htmx.NewComponent("templates/contributing-causes/binding/_form.html").FS(templates),
				bindContributingCausesOptions(ccs),
			),
			"BindContributingCause",
		).
		Attach("templates/reviews/__contributing-cause-bound-li.html").
		AddData("BoundContributingCauses", boundCauses).
		AddData("ReviewID", reviewID)
}
