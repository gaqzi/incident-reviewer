// This command is a bit gnarly and not very nicely factored,
// for now I'm fine with it since I doubt I'll change it very often.
// If someone wanted to pick up this pattern they should factor it nicer so it's clearer to read.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/sevlyar/go-daemon"

	"github.com/gaqzi/incident-reviewer/test"
)

var (
	postgresStartTimeout = 2 * time.Minute
	postgresUp           = false
	signal               = flag.String("s", "", `Send signal to the daemon:
  quit — graceful shutdown
  stop — fast shutdown`)
	stopChan = make(chan struct{})
	doneChan = make(chan struct{})

	// I need to tell the daemon to serve the healthcheck endpoint
	// at a known address to the parent, so I'll set it in an env variable
	// and read it in the child.
	healthcheckEnvName = "HEALTHCHECK_ADDR"
)

func main() {
	flag.Parse()
	daemon.AddCommand(daemon.StringFlag(signal, "quit"), syscall.SIGQUIT, termHandler)
	daemon.AddCommand(daemon.StringFlag(signal, "stop"), syscall.SIGTERM, termHandler)

	cntxt := &daemon.Context{
		PidFileName: "tmp/local-dev-dependencies.pid",
		PidFilePerm: 0644,
		LogFileName: "tmp/local-dev-dependencies.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"incident-reviewer__local-dev-dependencies"},
	}

	if len(daemon.ActiveFlags()) > 0 {
		d, err := cntxt.Search()
		if err != nil {
			log.Fatalf("Unable send signal to the daemon: %s", err.Error())
		}
		err = daemon.SendCommands(d)
		if err != nil {
			log.Fatalln(err.Error())
		}
		return
	}

	if len(os.Args) > 1 {
		if os.Args[1] != "stop" {
			log.Fatalf("unknown subcommand: %q", os.Args[1])
		}

		stop(cntxt)
	}

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalf("failed to create listener for the healthcheck")
	}
	healthcheckAddr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		log.Fatalf("failed to close listener: %s", err)
	}
	// Make HEALTHCHECK_ADDR and env var and make the daemon listen to that addr
	cntxt.Env = append(os.Environ(), fmt.Sprintf("%s=%s", healthcheckEnvName, healthcheckAddr))

	d, err := cntxt.Reborn()
	if err != nil {
		if errors.Is(err, daemon.ErrWouldBlock) {
			// I'm already running, so let's exit silently since there's nothing to do
			os.Exit(0)
		}
		log.Fatal("Unable to run: ", err)
	}

	// This block is only run in the parent / cli, and it will exit when it's done
	if d != nil {
		ctx, cancel := context.WithTimeout(context.Background(), postgresStartTimeout+2*time.Second)
		defer cancel()

		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/", healthcheckAddr), nil)
		if err != nil {
			log.Fatalf("failed to create http request: %s", err)
		}
		req.WithContext(ctx)

		for {
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				if strings.Contains(err.Error(), "connection refused") {
					time.Sleep(100 * time.Millisecond)
					continue
				}

				log.Fatalf("failed to call health check endpoint: %s", err)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("failed to read healthcheck body: %s", resp.Body)
			}

			if strings.HasSuffix(string(body), "true") {
				break
			}

			time.Sleep(100 * time.Millisecond)
		}

		return
	}

	// This code will only run in the child/daemon
	defer cntxt.Release()

	log.Print("- - - - - - - - - - - - - - -")
	log.Print("up and running")

	go serveHTTP()
	go startPostgres(stopChan, doneChan)

	err = daemon.ServeSignals()
	if err != nil {
		log.Printf("Error: %s", err.Error())
	}

	log.Println("daemon terminated")
}

func stop(cntxt *daemon.Context) {
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

	log.Printf("waiting for shutdown of local dev dependencies to complete")
	for {
		isAlive, err := cntxt.Search()
		if err != nil {
			log.Fatalf("error: %q", err)
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

func serveHTTP() {
	listenAddr := os.Getenv(healthcheckEnvName)
	if listenAddr == "" {
		log.Printf("didn't %s empty in env", healthcheckEnvName)
	}
	http.HandleFunc("/", httpHandler)
	log.Printf("about to listen to %q", listenAddr)
	http.ListenAndServe(listenAddr, nil)
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("request from %s: %s %q, postgresUp=%t", r.RemoteAddr, r.Method, r.URL, postgresUp)
	fmt.Fprintf(w, "postgresUp=%t", postgresUp)
}

func startPostgres(stopChan <-chan struct{}, doneChan chan<- struct{}) {
	// If 2min is not long enough for initial starts then change it, for now I'll guess it's good enough,
	// but that's from sitting with a stable (and fast) internet connection… If needed I'll make it configurable later.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	err, conn, done := test.StartPostgres(ctx)
	if err != nil {
		log.Fatalf("failed to start postgres: %s", err.Error())
	}

	f, err := os.Create("tmp/postgres.conf")
	if err != nil {
		log.Fatalf("failed to create postgres.conf: %s", err)
	}
	if _, err := f.WriteString(conn); err != nil {
		log.Fatalf("failed to write connection string to postgres.conf: %s", err)
	}
	cancel()

	postgresUp = true
	select {
	case <-stopChan:
		log.Printf("received stop signal")
		done()
		log.Printf("stopped postgres, time to report back")
		doneChan <- struct{}{}
	}
}

func termHandler(sig os.Signal) error {
	log.Println("terminating...")
	stopChan <- struct{}{}
	if sig == syscall.SIGQUIT {
		<-doneChan
	}
	return daemon.ErrStop
}
