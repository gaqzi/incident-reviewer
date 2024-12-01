package http

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"time"

	"github.com/donseba/go-htmx"
	"github.com/go-playground/form/v4"

	"github.com/gaqzi/incident-reviewer/internal/reviewing"
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

func Handler(store reviewing.Storage) *http.ServeMux {
	app := App{
		htmx:    htmx.New(),
		decoder: form.NewDecoder(),
		store:   store,
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.HandlerFunc(app.Index))
	mux.Handle("POST /", http.HandlerFunc(app.Create))

	return mux
}

func (a *App) Index(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	a.renderIndex(h, r, map[string]any{})
}

type IncidentBasic struct {
	URL         string `form:"url"         validate:"required,http_url"`
	Title       string `form:"title"       validate:"required"`
	Description string `form:"description" validate:"required"`
	Impact      string `form:"impact"      validate:"required"`
}

func (a *App) Create(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	if err := r.ParseForm(); err != nil {
		slog.Error("failed to parse form", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	var inc IncidentBasic
	if err := a.decoder.Decode(&inc, r.Form); err != nil {
		slog.Error("failed to decode basic incident form", "error", err)
		h.WriteHeader(http.StatusBadRequest)
		h.JustWriteString(err.Error())
		return
	}

	review := reviewing.Review{
		URL:         inc.URL,
		Title:       inc.Title,
		Description: inc.Description,
		Impact:      inc.Impact,
	}
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
		data["Reviews"] = reviews
	}

	page := htmx.NewComponent("templates/index.html").
		FS(templates).
		SetData(data).
		With(htmx.NewComponent("templates/_new.html").FS(templates), "New").
		Wrap(baseContent(), "Body")

	_, err := h.Render(r.Context(), page)
	if err != nil {
		slog.Error("failed to render page", "page", "reviews/index", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		_, _ = h.WriteString("failed to render")
	}
}

func baseContent() htmx.RenderableComponent {
	return htmx.NewComponent("templates/base.html").FS(templates)
}