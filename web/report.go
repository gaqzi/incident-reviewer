package web

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/donseba/go-htmx"
	"github.com/donseba/go-htmx/sse"
)

type App struct {
	htmx *htmx.HTMX
	sse  sse.Manager
}

type httpMux interface {
	Handle(pattern string, handler http.Handler)
}

func Register(mux httpMux) {
	app := &App{
		htmx: htmx.New(),
		sse:  sse.NewManager(5), // size decided by copying from the demo 😅
	}

	mux.Handle("GET /report/{code}", http.HandlerFunc(app.Show))
	mux.Handle("POST /report/{code}/root-causes", http.HandlerFunc(app.AddRootCauses))
	mux.Handle("GET /report/{code}/root-causes", http.HandlerFunc(app.ShowRootCauses))
	mux.Handle("/report/{code}/stream", http.HandlerFunc(app.Streaming))
}

func (a *App) Show(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	data := map[string]any{
		"Incident": map[string]any{
			"Code":        r.PathValue("code"),
			"Name":        "m000",
			"Description": "The Bangladesh government disabled internet due to student protests and 50% of customers were unable to connect",
		},
	}

	page := htmx.NewComponent("web/show.html").
		SetData(data).
		With(htmx.NewComponent("web/_root-causes.html"), "RootCauses").
		Wrap(baseContent(), "Body")

	_, err := h.Render(r.Context(), page)
	if err != nil {
		fmt.Printf("error rendering page: %v", err.Error())
	}
}

func (a *App) AddRootCauses(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	if h.RenderPartial() {
		if err := r.ParseForm(); err != nil {
			fmt.Printf("failed to parse form: %v", err)
			h.WriteHeader(http.StatusInternalServerError)
			return
		}

		data := map[string]any{
			"Incident": map[string]any{
				"Code": r.PathValue("code"),
				"AddRootCauses": []map[string]any{
					{"Type": r.PostForm.Get("type"), "Description": r.PostForm.Get("description")},
				},
			},
		}

		page := htmx.NewComponent("web/_root-causes.html").SetData(data)
		if _, err := h.Render(r.Context(), page); err != nil {
			fmt.Printf("error writing output: %v", err.Error())
		}
		a.sse.Send(sse.NewMessage("ping").WithEvent("root-causes"))

		return
	}

	w.Header().Set("Location", fmt.Sprintf("/report/%s", r.PathValue("code")))
	w.WriteHeader(http.StatusSeeOther)
}

func (a *App) Streaming(w http.ResponseWriter, r *http.Request) {
	client := sse.NewClient(randStringRunes(10))

	a.sse.Handle(w, r, client)
}

func (a *App) ShowRootCauses(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	data := map[string]any{
		"Incident": map[string]any{
			"Code": r.PathValue("code"),
			"AddRootCauses": []map[string]any{
				{"Type": "First", "Description": "A default description to give an example"},
			},
		},
	}

	page := htmx.NewComponent("web/_root-causes.html").SetData(data)
	if _, err := h.Render(r.Context(), page); err != nil {
		fmt.Printf("error writing output: %v", err.Error())
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func baseContent() htmx.RenderableComponent {
	return htmx.NewComponent("web/base.html")
}
