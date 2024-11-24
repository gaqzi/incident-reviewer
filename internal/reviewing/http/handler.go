package http

import (
	"embed"
	"log/slog"
	"net/http"

	"github.com/donseba/go-htmx"
)

//go:embed templates/*.html
var templates embed.FS

type App struct {
	htmx *htmx.HTMX
}

func Handler() *http.ServeMux {
	app := App{
		htmx: htmx.New(),
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.HandlerFunc(app.Index))

	return mux
}

func (a *App) Index(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	page := htmx.NewComponent("templates/index.html").
		FS(templates).
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
