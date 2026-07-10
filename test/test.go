package test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shahanmmiah/ravelpin/handlers"
	"github.com/shahanmmiah/ravelpin/internal/auth"
	"github.com/shahanmmiah/ravelpin/internal/database"
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
	"github.com/shahanmmiah/ravelpin/internal/services"
)

// LLAMA TEST

func TestLlama(cls *recoginition.ClassifyModels, imagePath string) error {
	img, err := recoginition.GetImageBytes(imagePath)

	err = cls.ClassifyImageDetails(img)
	//searchQueries, err := imageClasifier.SearchClassify.GetClasifyLabels(link, []string{"sweater", "pancho", "cardigan", "trousers", "jean", "sock", "sweatshirt", "mitten"})

	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("Error: %v", err)
	}

	//ynQueries, err := imageClasifier.YarnWeightClassify.GetClasifyLabels(link, nil)
	searchQueries := cls.LlammaResponseTable.GetOutputString(os.Getenv("LLAMMACLOTHTOOLNAME"))
	ynQueries := cls.LlammaResponseTable.GetOutputString(os.Getenv("LLAMMAYNTOOLNAME"))

	fmt.Printf("test yn found: %s\n test clothing type: %s", ynQueries, searchQueries)

	/*
		stream := false
		client, err := llama.ClientFromEnvironment()
		if err != nil {
			return fmt.Errorf("Error: %v", err)
		}
		req := &llama.GenerateRequest{
			Model:  os.Getenv("LLAMAMODEL"),
			Prompt: "tell me a joke.",
			//Images: []llama.ImageData{img},
			Stream: &stream,
		}

		respFunc := func(resp llama.GenerateResponse) error {

			fmt.Println(resp.Response)

			return nil
		}

		err = client.Generate(context.Background(), req, respFunc)
		if err != nil {
			return fmt.Errorf("Error: %v", err)
		}

		req = &llama.GenerateRequest{
			Model:  os.Getenv("LLAMAMODEL"),
			Prompt: "desibe whats in the image.",
			Images: []llama.ImageData{img},
			Stream: &stream,
		}

		err = client.Generate(context.Background(), req, respFunc)

		if err != nil {
			return fmt.Errorf("Error: %v", err)
		}
	*/

	return nil
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

func TestGetSearchPosts(cfg *handlers.ApiConfig, weights []services.YarnWeight, pageNum, pageSize int, wgtMap map[string]string, cmbMap map[string][]string) {

	idsChan := make(chan []int, len(weights))
	postChan := make(chan []services.RavelryPatternFull, len(weights))

	cfg.SearchYnRavelPosts(weights, pageNum, pageSize, idsChan, postChan, wgtMap, cmbMap)

}

func TestGetIdPosts(cfg *handlers.ApiConfig, start, inc int) []services.RavelryPatternFull {
	idsChan := make(chan []int)
	postChan := make(chan []services.RavelryPatternFull)

	go handlers.GetRavelIds(inc, start, idsChan)

	go handlers.GetRavelryPatternFull(idsChan, postChan)

	return <-postChan
}

func MakeYarnWeightDataset(cfg *handlers.ApiConfig) error {

	err := os.MkdirAll("./assets/datasets/yarnweights", 0755)
	if err != nil {
		fmt.Println(err)
		return err
	}
	weights, err := handlers.GetRavelYarnWeights()
	if err != nil {
		fmt.Println(err)
		return err
	}

	weightMap := make(map[string]string, 0)

	combineMap := map[string][]string{
		"Lace-Cobweb":     []string{"Lace", "Cobweb"},
		"Sport-Fingering": []string{"Sport", "Fingering", "Light-Fingering", "Thread"},
		"Aran-Worsted":    []string{"Aran", "Worsted"},
		"Bulky-Jumbo":     []string{"Bulky", "Super-Bulky", "Jumbo"},
	}
	for _, name := range weights {
		found := false
		for _, mp := range combineMap {
			for _, nme := range mp {
				grp := strings.ReplaceAll(name.Name, " ", "-")
				if grp == nme {
					found = true
				}
			}
		}
		if !found {
			grp := strings.ReplaceAll(name.Name, " ", "-")
			err := os.MkdirAll(fmt.Sprintf("./assets/datasets/yarnweights/%s", grp), 0755)
			if err != nil {
				fmt.Println(err)
				return err
			}
			weightMap[grp] = ""

		}
	}

	for grp, _ := range combineMap {
		err := os.MkdirAll(fmt.Sprintf("./assets/datasets/yarnweights/%s", grp), 0755)
		if err != nil {
			fmt.Println(err)
			return err
		}
		weightMap[grp] = ""

	}

	for num := range 1 {
		TestGetSearchPosts(cfg, weights, num+1, num+1*100, weightMap, combineMap)
	}

	return nil
}
