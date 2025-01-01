package http

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"go.uber.org/dig"
	"go.uber.org/zap"
	"net/http"
	"vault-injector/config"
	middleware "vault-injector/pkg/middlewere"
)

type WebServer interface {
	Start()
}

type simpleServer struct {
	server *http.Server
	ctx    context.Context
}

type ServerParams struct {
	dig.In
	config *config.Config //nolint
}

func NewWebServer(config *config.Config) WebServer {

	r := mux.NewRouter()

	r.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Ok")
	})
	r.HandleFunc("/readiness", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Ok")
	})
	r.HandleFunc("/liveness", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Ok")
	})

	r.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Stop NotImplements", http.StatusMethodNotAllowed)
	})
	r.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("Panic NotImplemented")
	})
	r.NotFoundHandler = r.NewRoute().HandlerFunc(http.NotFound).GetHandler()

	r.Use(middleware.AccessLog)
	r.Use(middleware.Panic)
	httpServer := &http.Server{
		Addr:    config.HTTP.ADDR,
		Handler: r,
	}

	return &simpleServer{
		server: httpServer,
	}
}

func (s *simpleServer) Start() {
	zap.S().Infof("starting server at %s", s.server.Addr)
	go s.server.ListenAndServe()
}

func adminIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, `<a href="/">site index</a>`)
	fmt.Fprintln(w, "Admin area")
}
