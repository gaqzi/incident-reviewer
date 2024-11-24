package http

import (
	"embed"
	"net/http"
)

//go:embed assets/*
var content embed.FS

// PublicAssets will register /assets/ and serve all assets in the ./assets folder.
func PublicAssets(mux *http.ServeMux) {
	mux.Handle(
		"GET /assets/",
		http.StripPrefix("/", http.FileServer(http.FS(content))),
	)
}
