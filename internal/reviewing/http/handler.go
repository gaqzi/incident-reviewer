package http

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/donseba/go-htmx"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/form/v4"
	"github.com/gosimple/slug"

	"github.com/gaqzi/incident-reviewer/internal/reviewing"
	"github.com/gaqzi/incident-reviewer/internal/reviewing/storage"
)

var (
	//go:embed templates/*
	templates embed.FS
)

type App struct {
	htmx    *htmx.HTMX
	decoder *form.Decoder
	store   reviewing.Storage
}

func Handler(store reviewing.Storage) func(chi.Router) {
	app := App{
		htmx:    htmx.New(),
		decoder: form.NewDecoder(),
		store:   store,
	}

	return func(r chi.Router) {
		r.Get("/", app.Index)
		r.Post("/", app.Create)
		r.Get("/{id}-{slug}", app.Show)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/edit", app.Edit)
			r.Post("/edit", app.Update)
		})
	}
}

func (a *App) Index(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	a.renderIndex(h, r, map[string]any{})
}

type ReviewBasic struct {
	ID                  int64  `form:"id"`
	URL                 string `form:"url"`
	Title               string `form:"title"`
	Description         string `form:"description"`
	Impact              string `form:"impact"`
	Where               string `form:"where"`
	ReportProximalCause string `form:"reportProximalCause"`
	ReportTrigger       string `form:"reportTrigger"`

	UpdatedAt time.Time
	CreatedAt time.Time
}

func (a *App) Create(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	if err := r.ParseForm(); err != nil {
		slog.Error("failed to parse form", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	var inc ReviewBasic
	if err := a.decoder.Decode(&inc, r.Form); err != nil {
		slog.Error("failed to decode basic incident form", "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString(err.Error())
		return
	}

	review := reviewing.Review{}
	assignFromHttpObject(inc, &review)
	rev, err := a.store.Save(r.Context(), review)
	if err != nil {
		slog.Error("failed to save incident", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		h.JustWriteString(err.Error())
		return
	}

	a.renderIndex(h, r, map[string]any{
		"New": map[string]any{
			"Created": map[string]any{
				"ID":    rev.ID,
				"Title": rev.Title,
			},
		},
	})
}

func (a *App) renderIndex(h *htmx.Handler, r *http.Request, data map[string]any) {
	if _, ok := data["Report"]; !ok {
		data["Report"] = map[string]any{}
	}
	if _, ok := data["Reviews"]; !ok {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		reviews, err := a.store.All(ctx)
		if err != nil {
			// Only log the error and set the empty listing as it's an okay fallback instead of returning an error
			slog.Error("failed to fetch all reviews", "error", err)
		}
		cancel()
		data["Reviews"] = convertToHttpObject(reviews)
	}

	page := htmx.NewComponent("templates/index.html").
		FS(templates).
		SetData(data).
		AddTemplateFunction("slug", slug.Make).
		With(
			htmx.NewComponent("templates/_new.html").
				FS(templates).
				Attach("templates/_review-fields.html"),
			"New",
		).
		Wrap(baseContent(), "Body")

	_, err := h.Render(r.Context(), page)
	if err != nil {
		slog.Error("failed to render page", "page", "reviews/index", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		_, _ = h.WriteString("failed to render")
	}
}

func (a *App) Show(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		slog.Error("failed to parse id for show", "id", r.PathValue("id"), "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString("invalid id")
		return
	}

	review, err := a.store.Get(r.Context(), reviewID)
	if err != nil {
		slog.Info("get review error", "error", err)

		var notFoundError *storage.NoReviewError
		if errors.As(err, &notFoundError) {
			h.WriteHeader(http.StatusNotFound)
			h.JustWriteString(fmt.Sprintf("404: review by id '%d' not found.", reviewID))
			return
		}
		slog.Error("error finding review", "error", err)
	}

	data := map[string]any{
		"Review": review,
	}

	page := htmx.NewComponent("templates/show.html").
		FS(templates).
		SetData(data).
		AddTemplateFunction("slug", slug.Make).
		Wrap(baseContent(), "Body")

	_, err = h.Render(r.Context(), page)
	if err != nil {
		slog.Error("failed to render", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		_, _ = h.WriteString("failed to render")
	}
}

func (a *App) Edit(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		slog.Error("failed to parse id for show", "id", r.PathValue("id"), "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString("invalid id")
		return
	}

	review, err := a.store.Get(r.Context(), reviewID)
	if err != nil {
		slog.Info("get review error", "error", err)

		var notFoundError *storage.NoReviewError
		if errors.As(err, &notFoundError) {
			h.WriteHeader(http.StatusNotFound)
			h.JustWriteString(fmt.Sprintf("404: review by id '%d' not found.", reviewID))
			return
		}
		slog.Error("error finding review", "error", err)
	}

	data := map[string]any{
		"Review": review,
	}

	page := htmx.NewComponent("templates/edit.html").
		FS(templates).
		SetData(data).
		Attach("templates/_review-fields.html").
		AddTemplateFunction("slug", slug.Make).
		Wrap(baseContent(), "Body")

	_, err = h.Render(r.Context(), page)
	if err != nil {
		slog.Error("failed to render", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		_, _ = h.WriteString("failed to render")
	}

}

func (a *App) Update(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		slog.Error("failed to parse id for show", "id", r.PathValue("id"), "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString("invalid id")
		return
	}

	review, err := a.store.Get(r.Context(), reviewID)
	if err != nil {
		slog.Info("get review error", "error", err)

		var notFoundError *storage.NoReviewError
		if errors.As(err, &notFoundError) {
			h.WriteHeader(http.StatusNotFound)
			h.JustWriteString(fmt.Sprintf("404: review by id '%d' not found.", reviewID))
			return
		}
		slog.Error("error finding review", "error", err)
	}

	if err := r.ParseForm(); err != nil {
		slog.Error("failed to parse form", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	var inc ReviewBasic
	if err := a.decoder.Decode(&inc, r.Form); err != nil {
		slog.Error("failed to decode basic incident form", "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString(err.Error())
		return
	}

	// Update all the fields that can change. Note: We don't change CreatedAt
	assignFromHttpObject(inc, &review)

	_, err = a.store.Save(r.Context(), review)
	if err != nil {
		slog.Error("failed to save review", "id", reviewID, "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		h.JustWriteString(err.Error())
		return
	}

	h.Header().Add("Location", fmt.Sprintf("/reviews/%d-%s", reviewID, slug.Make(inc.Title)))
	h.WriteHeader(http.StatusSeeOther)
}

func baseContent() htmx.RenderableComponent {
	return htmx.NewComponent("templates/base.html").FS(templates)
}

func convertToHttpObject(rs []reviewing.Review) []ReviewBasic {
	ret := make([]ReviewBasic, 0, len(rs))

	for _, r := range rs {
		ret = append(ret, ReviewBasic{
			ID:                  r.ID,
			URL:                 r.URL,
			Title:               r.Title,
			Description:         r.Description,
			Impact:              r.Impact,
			Where:               r.Where,
			ReportProximalCause: r.ReportProximalCause,
			ReportTrigger:       r.ReportTrigger,

			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		})
	}

	return ret
}

// assignFromHttpObject takes all values from rb and assigns them to r.
func assignFromHttpObject(rb ReviewBasic, r *reviewing.Review) {
	r.URL = rb.URL
	r.Title = rb.Title
	r.Description = rb.Description
	r.Impact = rb.Impact
	r.Where = rb.Where
	r.ReportProximalCause = rb.ReportProximalCause
	r.ReportTrigger = rb.ReportTrigger
}
