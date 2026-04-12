package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/shahanmmiah/ravelpin/internal/database"
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
	"github.com/shahanmmiah/ravelpin/internal/services"
)

const THREASHOLD = 5

func (cfg *ApiConfig) AddRavelhash(posts []services.RavelryPattern) {

	for _, pst := range posts {
		err := cfg.AddRavelPostHash(pst)
		if err != nil {
			fmt.Printf("error adding post hash %s", err.Error())
		}

	}
}

func (cfg *ApiConfig) AddRavelPostHash(pattern services.RavelryPattern) error {

	imgHash, err := recoginition.CreatePHash(pattern.FirstPhoto.MediumURL)
	if err != nil {
		return err
	}

	subHashes := recoginition.Split16BitHash(imgHash.GetHash())

	postDb, err := cfg.Db.GetRavelHashes(context.Background(), database.GetRavelHashesParams{

		HashPart1: subHashes[0],
		HashPart2: subHashes[1],
		HashPart3: subHashes[2],
		HashPart4: subHashes[3],
	})

	if len(postDb) > 0 {
		return fmt.Errorf("ravel post already in database")
	}

	_, err = cfg.Db.CreateRavelHash(context.Background(), database.CreateRavelHashParams{
		ImagePath: pattern.FirstPhoto.MediumURL,
		RavelPost: fmt.Sprintf("%v%s", os.Getenv("RAVELURL"), pattern.Permalink),
		FullHash:  int64(imgHash.GetHash()),
		HashPart1: subHashes[0],
		HashPart2: subHashes[1],
		HashPart3: subHashes[2],
		HashPart4: subHashes[3],
		RavelID:   int32(pattern.Id),
		Permalink: pattern.Permalink,
		PostName:  pattern.Name,
	})

	if err != nil {
		return err
	}

	return nil

}

func (cfg *ApiConfig) GetRavelHashes(imagePath string, amount int) ([]services.RavelryPattern, error) {

	imgHash, err := recoginition.CreatePHash(imagePath)
	if err != nil {
		return nil, err
	}

	slog.Info(fmt.Sprintf("imageHash %v \n", imgHash))
	subHashes := recoginition.Split16BitHash(imgHash.GetHash())

	postDb, err := cfg.Db.GetRavelHashes(context.Background(), database.GetRavelHashesParams{

		HashPart1: subHashes[0],
		HashPart2: subHashes[1],
		HashPart3: subHashes[2],
		HashPart4: subHashes[3],
	})

	slog.Info(fmt.Sprintf("PostDb %v \n", postDb))

	if err != nil {
		return nil, fmt.Errorf("Error getting ravelpost hashes %s", err.Error())
	}

	bestPatterns := make([]services.RavelryPattern, 0)

	if len(postDb) < amount {
		amount = len(postDb)

	}

	for num := range amount {

		dst, err := imgHash.Distance(recoginition.HashFromInt(postDb[num].FullHash))
		if err == nil && dst < THREASHOLD {
			slog.Info(fmt.Sprintf("%s - distance is %v ", postDb[num].RavelPost, dst))

			pattern := services.RavelryPattern{
				Id:         int(postDb[num].RavelID),
				Name:       postDb[num].PostName,
				Permalink:  postDb[num].Permalink,
				FirstPhoto: services.RavelPhoto{MediumURL: postDb[num].ImagePath}}

			bestPatterns = append(bestPatterns, pattern)

		}

	}

	return bestPatterns, nil

}
