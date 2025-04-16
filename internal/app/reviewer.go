package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/google/uuid"

	"github.com/gaqzi/incident-reviewer/internal/app/web"
	"github.com/gaqzi/incident-reviewer/internal/normalized"
	"github.com/gaqzi/incident-reviewer/internal/normalized/contributing"
	contribstorage "github.com/gaqzi/incident-reviewer/internal/normalized/contributing/storage"
	"github.com/gaqzi/incident-reviewer/internal/normalized/storage"
	"github.com/gaqzi/incident-reviewer/internal/reviewing"
	reviewstorage "github.com/gaqzi/incident-reviewer/internal/reviewing/storage"
)

type Config struct {
	Addr string
}

func NewConfig() Config {
	return Config{
		Addr: "127.0.0.1:3000",
	}
}

type Server struct {
	Config Config
	HTTP   *http.Server
}

// Stop will shut down the server safely.
func (s *Server) Stop(ctx context.Context) error {
	return s.HTTP.Shutdown(ctx)
}

// Start wires up the app and starts running it
func Start(ctx context.Context, cfg Config) (*Server, error) {
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen to %q: %w", cfg.Addr, err)
	}
	cfg.Addr = ln.Addr().String() // In case cfg.Addr was random we'll update the config to point to what we ended up using

	server := http.Server{}
	server.BaseContext = func(_ net.Listener) context.Context { return ctx }
	r := chi.NewRouter()
	server.Handler = r

	logger := httplog.NewLogger("incident-reviewer", httplog.Options{
		// JSON:           true,
		LogLevel:       slog.LevelInfo,
		Concise:        true,
		RequestHeaders: true,
		QuietDownRoutes: []string{
			"/",
			"/favicon.ico",
		},
		QuietDownPeriod: 10 * time.Second,
	})

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(httplog.RequestLogger(logger))
	r.Use(middleware.Recoverer)

	web.PublicAssets(r)

	causeService := contributing.NewCauseService(contribstorage.NewCauseMemoryStore())
	cause := contributing.NewCause()
	cause.Name = "Third party outage"
	cause.Description = "In case a third party experienced issues/outage and it leads to an incident on our side.\nThings like third party changing configuration and it leading to issues on our side also qualifies"
	cause.Category = "Design"
	_, err = causeService.Save(ctx, cause)
	if err != nil {
		return nil, fmt.Errorf("failed to add default contributing causes: %w", err)
	}
	r.Route("/contributing-causes", web.ContributingCausesHandler(causeService))

	reviewStore := reviewstorage.NewMemoryStore()

	triggerService := normalized.NewTriggerService(storage.NewTriggerMemoryStore())
	trigger := normalized.Trigger{}
	trigger.ID = uuid.MustParse("6A195282-04CA-4405-A6F1-678C525A001B")
	trigger.Name = "Traffic increase"
	trigger.Description = "More users than normal"
	_, err = triggerService.Save(ctx, trigger)
	if err != nil {
		return nil, fmt.Errorf("failed to add default trigger: %w", err)
	}

	reviewService := reviewing.NewService(reviewStore, causeService, triggerService)
	r.Route("/reviews", web.ReviewsHandler(reviewService, causeService))

	go (func() {
		_ = server.Serve(ln)
	})()

	return &Server{
		Config: cfg,
		HTTP:   &server,
	}, nil
}
