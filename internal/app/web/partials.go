package web

import "github.com/donseba/go-htmx"

func bindContributingCause() (htmx.RenderableComponent, string) {
	return htmx.NewComponent("templates/contributing-causes/binding/_form.html").
			FS(templates).
			Attach("templates/contributing-causes/binding/_causes-options.html"),
		"BindContributingCause"
}
