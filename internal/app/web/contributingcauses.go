package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/donseba/go-htmx"
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
}

func ContributingCausesHandler(service causeService) func(chi.Router) {
	a := causesHandler{
		htmx:    htmx.New(),
		service: service,
	}

	return func(r chi.Router) {
		r.Post("/", a.Create)
		r.Get("/new", a.New)
	}
}

func (a *causesHandler) New(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	newForm := htmx.NewComponent("templates/contributing-causes/new.html").
		FS(templates).
		Attach("templates/contributing-causes/_fields.html")

	if !h.IsHxRequest() {
		h.WriteHeader(http.StatusNotFound)
		h.JustWriteString("not yet supported")
	}

	_, err := h.Render(r.Context(), newForm)
	if err != nil {
		slog.Error("failed to render new form", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		h.JustWriteString("failed to render")
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

	page := attachToComponent(
		htmx.NewComponent("templates/contributing-causes/binding/only-options.html").
			FS(templates).
			AddData("SelectedCauseID", cause.ID.String()),
		bindContributingCausesOptions(causes),
	)

	_, err = h.Render(r.Context(), page)
	if err != nil {
		slog.Error("failed to render partial contributing-causes/binding/", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}
}
