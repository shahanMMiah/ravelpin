package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/shahanmmiah/ravelpin/components"
	"github.com/shahanmmiah/ravelpin/handlers"
	"github.com/shahanmmiah/ravelpin/internal/database"
	"github.com/shahanmmiah/ravelpin/internal/ratelimit"
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
)

type Server struct {
	Mux       *http.ServeMux
	RateLimit *ratelimit.RateLimiter
}

type ApiConfig struct {
	Db   *database.Queries
	Serv Server
}

func (serv *Server) limitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		if !serv.RateLimit.GetClientRateLimit(ip) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too Many Requests"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (cfg *ApiConfig) SetupServer(clasifyModel *recoginition.ClassifyModel) {

	htmxHandle := handlers.MiddleWareServeFile("./assets/js/htmx.min.js")
	handlers.HandleHandler(cfg.Serv.Mux, htmxHandle, "/assets/js/htmx.min.js", "GET")

	component := components.HomePage()

	cfg.Serv.Mux.Handle("/{$}", templ.Handler(component))

	slog.Info("Listening on :3000")

	err := handlers.HandleHandler(cfg.Serv.Mux, cfg.Serv.limitMiddleware(handlers.MiddleWareGetRavelLink(clasifyModel)), "/link", "POST")
	if err != nil {
		log.Panicf("error handler %v", err.Error())
	}

	logsHandler := handlers.HandlerGetLogs(cfg.Serv.Mux)

	cfg.Serv.Mux.Handle("/logs", cfg.Serv.limitMiddleware(logsHandler))

	err = handlers.HandleHandler(cfg.Serv.Mux, cfg.Serv.limitMiddleware(handlers.HandlerGetLog()), "/log", "POST")
	if err != nil {
		log.Panicf("error handler %v", err.Error())
	}

	server := http.Server{
		Handler: cfg.Serv.Mux,
		Addr:    ":3000",
	}

	server.ListenAndServe()

}

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	slog.SetDefault(logger)

	godotenv.Load()

	db, err := sql.Open("postgres", os.Getenv("DBPATH"))

	if err != nil {
		slog.ErrorContext(context.Background(), fmt.Sprintf("error loading database: %v", err.Error()))
		os.Exit(1)
	}

	cfg := ApiConfig{Db: database.New(db), Serv: Server{
		Mux:       http.NewServeMux(),
		RateLimit: ratelimit.NewRateLimiter(),
	}}

	imageClassifyModel := recoginition.NewModel()

	imageClassifyModel.LoadModel(fmt.Sprintf("./model/%s", os.Getenv("CLASSIFYMODELNAME")))

	cfg.SetupServer(imageClassifyModel)

}
