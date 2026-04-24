package services

import (
	"sort"

	"github.com/shahanmmiah/ravelpin/internal/recoginition"
)

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
	sort.Sort(matches)
	bestPatterns := make([]RavelryPattern, 0)

	if len(matches) < postAmount {
		postAmount = len(matches)

	}
	for num := range postAmount {
		bestPatterns = append(bestPatterns, matches[num].ID.(RavelryPattern))
	}

	return bestPatterns, nil

}
