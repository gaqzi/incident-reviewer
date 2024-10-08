package main

import (
	"log"
	"net/http"

	"github.com/donseba/go-htmx"

	"github.com/gaqzi/incident-reviewer/web"
)

type App struct {
	htmx *htmx.HTMX
}

func main() {
	// new app with htmx instance
	// app := &App{
	// 	htmx: htmx.New(),
	// }

	mux := http.NewServeMux()
	mux.Handle(
		"GET /static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("./public"))),
	)
	web.Register(mux)

	err := http.ListenAndServe(":3000", mux)
	log.Fatal(err)
}

func (a *App) Home(w http.ResponseWriter, r *http.Request) {
	// initiate a new htmx handler
	h := a.htmx.NewHandler(w, r)

	// check if the request is a htmx request
	if h.IsHxRequest() {
		// do something
	}

	// check if the request is boosted
	if h.IsHxBoosted() {
		// do something
	}

	// check if the request is a history restore request
	if h.IsHxHistoryRestoreRequest() {
		// do something
	}

	// check if the request is a prompt request
	if h.RenderPartial() {
		// do something
	}

	// set the headers for the response, see docs for more options
	h.PushURL("http://push.url")
	h.ReTarget("#ReTarged")

	// write the output like you normally do.
	// check the inspector tool in the browser to see that the headers are set.
	_, _ = h.Write([]byte("OK"))
}
