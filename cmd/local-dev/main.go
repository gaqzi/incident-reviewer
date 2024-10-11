package main

import (
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sevlyar/go-daemon"
)

// To terminate the daemon use:
//
//	kill `cat sample.pid`
func main() {
	cntxt := &daemon.Context{
		PidFileName: "tmp/local-dev.pid",
		PidFilePerm: 0644,
		LogFileName: "tmp/local-dev.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"local-dev"},
	}

	if len(os.Args) > 1 {
		if os.Args[1] != "stop" {
			log.Fatalf("unknown subcommand: %q", os.Args[1])
		}

		proc, err := cntxt.Search()
		if err != nil {
			log.Fatalf("failed to find process: %s", err.Error())
		}
		// process isn't alive, so don't try anything more
		if proc == nil {
			os.Exit(0)
		}

		if err := proc.Kill(); err != nil {
			log.Fatalf("failed to kill process: %s", err.Error())
		}

		log.Printf("waiting for shutdown to complete")
		for {
			isAlive, err := cntxt.Search()
			if err != nil {
				log.Fatal("error: %q", err)
			}
			// no process found so let's assume it's ded now
			if isAlive == nil {
				fmt.Print("\n")
				os.Exit(0)
			}
			fmt.Print(".")
			time.Sleep(100 * time.Millisecond)
		}
	}

	d, err := cntxt.Reborn()
	if err != nil {
		if errors.Is(err, daemon.ErrWouldBlock) {
			log.Print("already running")
			os.Exit(0)
		}
		log.Fatal("Unable to run: ", err)
	}
	if d != nil {
		log.Println("parent quitting")
		return
	}
	defer cntxt.Release()

	log.Print("- - - - - - - - - - - - - - -")
	log.Print("daemon started")

	serveHTTP()
}

func serveHTTP() {
	http.HandleFunc("/", httpHandler)
	http.ListenAndServe("127.0.0.1:8080", nil)
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("request from %s: %s %q", r.RemoteAddr, r.Method, r.URL)
	fmt.Fprintf(w, "go-daemon: %q", html.EscapeString(r.URL.Path))
}
