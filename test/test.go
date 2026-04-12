package test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/shahanmmiah/ravelpin/handlers"
	"github.com/shahanmmiah/ravelpin/internal/auth"
	"github.com/shahanmmiah/ravelpin/internal/database"
	"github.com/shahanmmiah/ravelpin/internal/services"
)

func TestAddRavelhash(cfg *handlers.ApiConfig, posts []services.RavelryPattern) {

	for _, pst := range posts {
		err := cfg.AddRavelPostHash(pst)
		if err != nil {
			fmt.Printf("error adding post hash %s", err.Error())
		}

	}

}

func TestAddAndGetRavelHash(cfg *handlers.ApiConfig) {

	posts, err := handlers.GetRavelPatterns([]string{"florence miller"}, 20, 1)
	if err != nil {
		fmt.Printf("error getting posts %s", err.Error())
	}

	TestAddRavelhash(cfg, posts)

	hashs, err := cfg.GetRavelHashes(posts[0].FirstPhoto.MediumURL, 10)
	if err != nil {
		fmt.Printf("error getting hashs %s", err.Error())
	}

	fmt.Printf("hashs posts found %v", hashs)

}

func ravelParamTest() {

	APIUS := os.Getenv("RAVELRYAPIUS")
	APIKEY := os.Getenv("RAVELRYAPIKEY")

	url := "https://api.ravelry.com/pattern_attributes/groups.json"

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s", url), nil)
	if err != nil {
		log.Print("client: could not create request", err)
		os.Exit(1)
	}
	req.SetBasicAuth(APIUS, APIKEY)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Print("client: could not create request", err)
		os.Exit(1)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)

	if err != nil {
		log.Print("client: could not create request", err)
		os.Exit(1)
	}

	log.Println(string(data))

}

func TestMakeAdminUser(cfg *handlers.ApiConfig) {
	password, _ := auth.HashPassword("admin")

	_, err := cfg.Db.CreateUser(context.Background(),
		database.CreateUserParams{
			ID:             uuid.New(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			Email:          "admin@ravelpin.com",
			HashedPassword: password})

	if err != nil {
		panic(fmt.Sprintf("error creating user %v", err))
	}

}
