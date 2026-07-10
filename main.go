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
	llama "github.com/ollama/ollama/api"
	"github.com/shahanmmiah/ravelpin/handlers"
	"github.com/shahanmmiah/ravelpin/internal/database"
	"github.com/shahanmmiah/ravelpin/internal/logging"
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
		SSE:       logging.NewSSE(),
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

	// init classify models
	searchClassifyModel := recoginition.NewModel()
	searchClassifyModel.LoadModel(fmt.Sprintf("./model/%s", os.Getenv("SEARCHCLASSIFYMODELNAME")), "serving_default_inputs")

	ynClassifyModel := recoginition.NewModel()
	ynClassifyModel.LoadModel(fmt.Sprintf("./model/%s", os.Getenv("YNCLASSIFYMODELNAME")), "serving_default_sequential_input")

	llamaClient, err := recoginition.NewLlamaClient()
	if err != nil {
		slog.ErrorContext(context.Background(), fmt.Sprintf("error init llama model: %v", err.Error()))
		os.Exit(1)
	}

	classifyModels := &recoginition.ClassifyModels{
		YarnWeightClassify:  ynClassifyModel,
		SearchClassify:      searchClassifyModel,
		LlammaClient:        llamaClient,
		LlammaToolFuncs:     llama.Tools{recoginition.NewYnTool(), recoginition.NewClothingTool()},
		LlammaStream:        false,
		LlammaResponseTable: recoginition.NewResponseMap(),
	}

	// TESTS

	test.TestMakeAdminUser(&cfg)

	//test.TestAddAndGetRavelHash(&cfg)

	//test.TestGetRavelIdPattern(&cfg)

	//test.TestGetRavelIds()

	//cfg.GatherRavelPosts(10, 50)

	/*
		err = test.MakeYarnWeightDataset(&cfg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	*/

	//test.TestLlama(classifyModels, "https://images4-f.ravelrycache.com/flickr/2/9/9/2999083585/2999083585.jpg")

	cfg.SetupServer(classifyModels)

}
