package services

import (
	"github.com/shahanmmiah/ravelpin/internal/recoginition"
)

const SCORETHRSHOLD = 500.0

func GetBestsImages(patterns []RavelryPattern, trgpath string, postAmount int) ([]RavelryPattern, error) {

	store := recoginition.CreateStore()

	trgHash, err := recoginition.CreateHash(trgpath)
	if err != nil {
		return []RavelryPattern{}, err

	}

	for _, pattern := range patterns {

		recoginition.AddToStore(store, pattern, pattern.FirstPhoto.MediumURL)

	}

	matches := store.Query(trgHash)
	//sort.Sort(matches)
	bestPatterns := make([]RavelryPattern, 0)

	for _, m := range matches {

		if m.Score < SCORETHRSHOLD && len(bestPatterns) <= postAmount {
			bestPatterns = append(bestPatterns, m.ID.(RavelryPattern))

		}
	}

	/*

		if len(matches) < postAmount {
			postAmount = len(matches)

		}
		for num := range postAmount {
			bestPatterns = append(bestPatterns, matches[num].ID.(RavelryPattern))
		}
	*/

	return bestPatterns, nil

}
