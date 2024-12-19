package web

import (
	"embed"
	"net/http"
)

//go:embed assets/*
var content embed.FS

type getRoutabler interface {
	Handle(string, http.Handler)
}

// PublicAssets will register /assets/ and serve all assets in the ./assets folder.
func PublicAssets(mux getRoutabler) {
	mux.Handle(
		"/assets/*",
		http.StripPrefix("/", http.FileServer(http.FS(content))),
	)
}
