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
	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/normalized"
)

// TriggerBasic is a simplified version of normalized.Trigger for use in templates.
type TriggerBasic struct {
	ID          uuid.UUID
	Name        string
	Description string
}

// convertTriggersToHttpObjects converts a slice of normalized.Trigger to a slice of TriggerBasic.
func convertTriggersToHttpObjects(triggers []normalized.Trigger) []TriggerBasic {
	ret := make([]TriggerBasic, 0, len(triggers))
	for _, t := range triggers {
		ret = append(ret, TriggerBasic{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
		})
	}
	return ret
}

type triggerService interface {
	Save(ctx context.Context, trigger normalized.Trigger) (normalized.Trigger, error)
	All(ctx context.Context) ([]normalized.Trigger, error)
}

type triggersHandler struct {
	htmx    *htmx.HTMX
	service triggerService
	partial *partial.Service
	pp      *passepartout.Passepartout
}

func TriggersHandler(service triggerService) func(chi.Router) {
	fsys, err := passepartout.FSWithoutPrefix(templates, "templates")
	if err != nil {
		panic(err)
	}

	partials := &ppdefaults.PartialsWithCommon{FS: fsys, CommonDir: "partials"}
	a := triggersHandler{
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

func (a *triggersHandler) New(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	if !h.IsHxRequest() {
		h.WriteHeader(http.StatusNotFound)
		h.JustWriteString("not yet supported")
		return
	}

	if err := a.pp.Render(w, "triggers/new.html", nil); err != nil {
		slog.Error("failed to render new form", "error", err)
		http.Error(w, "failed to render", http.StatusInternalServerError)
		return
	}
}

func (a *triggersHandler) Create(w http.ResponseWriter, r *http.Request) {
	h := a.htmx.NewHandler(w, r)

	if !h.IsHxRequest() {
		h.WriteHeader(http.StatusNotFound)
		h.JustWriteString("not yet supported")
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.Error("failed to parse form", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	trigger := normalized.NewTrigger()
	trigger.Name = r.PostForm.Get("name")
	trigger.Description = r.PostForm.Get("description")

	trigger, err := a.service.Save(r.Context(), trigger)
	if err != nil {
		slog.Error("failed to save new trigger", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	triggers, err := a.service.All(r.Context())
	if err != nil {
		slog.Error("failed to fetch all triggers after proposing new trigger", "error", err)
		h.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"SelectedTriggerID": trigger.ID.String(),
		"Triggers":          convertTriggersToHttpObjects(triggers),
	}

	if err := a.pp.Render(w, "triggers/new/_options.html", data); err != nil {
		slog.Error("failed to render partial triggers/new/_options.html", "error", err)
		http.Error(w, "failed to render", http.StatusInternalServerError)
		return
	}
}
