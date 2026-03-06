package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/shahanmmiah/ravelpin/handlers"
	"github.com/shahanmmiah/ravelpin/internal/database"
	"github.com/shahanmmiah/ravelpin/internal/ratelimit"
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
)

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	slog.SetDefault(logger)

	godotenv.Load()

	db, err := sql.Open("postgres", os.Getenv("DBPATH"))

	if err != nil {
		slog.ErrorContext(context.Background(), fmt.Sprintf("error loading database: %v", err.Error()))
		os.Exit(1)
	}

	cfg := handlers.ApiConfig{Db: database.New(db), Serv: handlers.Server{
		Mux:       http.NewServeMux(),
		RateLimit: ratelimit.NewRateLimiter(),
	}}

	//test.TestMakeAdminUser(&cfg)

	imageClassifyModel := recoginition.NewModel()

	imageClassifyModel.LoadModel(fmt.Sprintf("./model/%s", os.Getenv("CLASSIFYMODELNAME")))

	cfg.SetupServer(imageClassifyModel)

}
