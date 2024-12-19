package http

import (
	"context"
	"embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/donseba/go-htmx"
	"github.com/go-chi/chi/v5"

	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
)

var (
	//go:embed all:templates/*
	templates embed.FS
)

type causeService interface {
	Save(ctx context.Context, cause contributing.Cause) (contributing.Cause, error)
}

type app struct {
	htmx    *htmx.HTMX
	service causeService
}

func Handler(service causeService) func(chi.Router) {
	a := app{
		htmx:    htmx.New(),
		service: service,
	}

	return func(r chi.Router) {
		r.Post("/", a.Create)
		r.Get("/new", a.New)
	}
}

func (a *app) New(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	newForm := htmx.NewComponent("templates/new.html").
		FS(templates).
		Attach("templates/_fields.html").
		AddData("ReturnTo", r.Header.Get("hx-current-url"))

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

func (a *app) Create(w http.ResponseWriter, r *http.Request) {
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

	target := r.Header.Get("hx-target")
	location := r.Header.Get("hx-current-url")
	locationURL, err := url.Parse(location)
	if err != nil {
		slog.Error("failed to parse location URL for cause success creation redirect", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}
	q := locationURL.Query()
	q.Add("selectedCause", cause.ID.String())
	locationURL.RawQuery = q.Encode()
	location = locationURL.String()
	redirect := map[string]string{
		"target": "#" + target,
		"select": "#" + target,
		"path":   location,
	}
	slog.Info("preparing to redirect", "url", location)
	data, err := json.Marshal(redirect)
	if err != nil {
		slog.Error("failed to encode json for cause success creation redirect", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.Header().Add(string(htmx.HXLocation), string(data))
	h.WriteHeader(http.StatusCreated) // TODO: handle when it's not HTMX boosted
}
