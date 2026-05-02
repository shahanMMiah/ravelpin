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
	"github.com/shahanmmiah/ravelpin/test"
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

	err = cfg.Db.ResetRefreshToken(context.Background())
	if err != nil {
		slog.ErrorContext(context.Background(), fmt.Sprintf("error resetting refresh tokens: %v", err.Error()))
		os.Exit(1)
	}

	err = cfg.Db.ResetUsers(context.TODO())
	if err != nil {
		slog.ErrorContext(context.Background(), fmt.Sprintf("error resetting user database: %v", err.Error()))
		os.Exit(1)
	}

	err = cfg.Db.ResetVerifyTokens(context.TODO())
	if err != nil {
		slog.ErrorContext(context.Background(), fmt.Sprintf("error resetting verify tokens database: %v", err.Error()))
		os.Exit(1)
	}

	err = cfg.Db.ResetRavelHashes(context.TODO())
	if err != nil {
		slog.ErrorContext(context.Background(), fmt.Sprintf("error resetting ravel hash database: %v", err.Error()))
		os.Exit(1)
	}

	// TESTS

	test.TestMakeAdminUser(&cfg)

	//test.TestAddAndGetRavelHash(&cfg)

	//test.TestGetRavelIdPattern(&cfg)

	//test.TestGetRavelIds()

	cfg.GatherRavelPosts(10, 50)
	imageClassifyModel := recoginition.NewModel()

	imageClassifyModel.LoadModel(fmt.Sprintf("./model/%s", os.Getenv("CLASSIFYMODELNAME")))

	cfg.SetupServer(imageClassifyModel)

}
