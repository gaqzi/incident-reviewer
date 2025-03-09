package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/donseba/go-htmx"
	"github.com/donseba/go-partial"
	"github.com/donseba/go-partial/connector"
	"github.com/gaqzi/passepartout"
	"github.com/gaqzi/passepartout/ppdefaults"
	"github.com/go-chi/chi/v5"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
)

type causeService interface {
	Save(ctx context.Context, cause contributing.Cause) (contributing.Cause, error)
	All(ctx context.Context) ([]contributing.Cause, error)
}

type causesHandler struct {
	htmx    *htmx.HTMX
	service causeService
	partial *partial.Service
	pp      *passepartout.Passepartout
}

func ContributingCausesHandler(service causeService) func(chi.Router) {
	fsys, err := passepartout.FSWithoutPrefix(templates, "templates")
	if err != nil {
		panic(err)
	}

	partials := &ppdefaults.PartialsWithCommon{FS: fsys, CommonDir: "partials"}
	a := causesHandler{
		htmx:    htmx.New(),
		service: service,
		pp: passepartout.New(
			ppdefaults.NewLoaderBuilder().
				WithDefaults(fsys).
				TemplateLoader(ppdefaults.NewCachedLoader(&ppdefaults.TemplateByNameLoader{FS: fsys})).
				PartialsFor(partials.Load).
				Build(),
		),
	}

	partialConf := partial.Config{
		Connector: connector.NewHTMX(&connector.Config{
			UseURLQuery: true, // Allow fallback to URL query parameters (???)
		}),
		UseCache: true,
	}
	a.partial = partial.NewService(&partialConf)

	return func(r chi.Router) {
		r.Post("/", a.Create)
		r.Get("/new", a.New)
	}
}

func (a *causesHandler) New(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	if !h.IsHxRequest() {
		h.WriteHeader(http.StatusNotFound)
		h.JustWriteString("not yet supported")
	}

	if err := a.pp.Render(w, "contributing-causes/new.html", nil); err != nil {
		slog.Error("failed to render new form", "error", err)
		http.Error(w, "failed to render", http.StatusInternalServerError)
		return
	}
}

func (a *causesHandler) Create(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	if !h.IsHxRequest() {
		h.WriteHeader(http.StatusNotFound)
		h.JustWriteString("not yet supported")
	}

	if err := r.ParseForm(); err != nil {
		slog.Error("failed to parse form", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	cause := contributing.NewCause()
	cause.Name = r.PostForm.Get("name")
	cause.Description = r.PostForm.Get("description")
	cause.Category = r.PostForm.Get("category")

	cause, err := a.service.Save(r.Context(), cause)
	if err != nil {
		slog.Error("failed to save new contributing cause", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	causes, err := a.service.All(r.Context())
	if err != nil {
		slog.Error("failed to fetch all contributing causes after proposing new cause", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"SelectedCauseID":    cause.ID.String(),
		"ContributingCauses": convertContributingCauseToHttpObjects(causes),
	}

	if err := a.pp.Render(w, "contributing-causes/new/_options.html", data); err != nil {
		slog.Error("failed to render partial contributing-causes/binding/", "error", err)
		http.Error(w, "failed to render", http.StatusInternalServerError)
		return
	}
}
