package app

import (
	"context"
	"fmt"
	"net"
	"net/http"

	httpassets "github.com/gaqzi/incident-reviewer/internal/platform/http"
	revhttp "github.com/gaqzi/incident-reviewer/internal/reviewing/http"
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
	mux := http.NewServeMux()
	server.Handler = mux

	httpassets.PublicAssets(mux)
	reviewStore := reviewstorage.NewMemoryStore()
	mux.Handle("/reviews", revhttp.Handler(reviewStore))

	go (func() {
		_ = server.Serve(ln)
	})()

	return &Server{
		Config: cfg,
		HTTP:   &server,
	}, nil
}
