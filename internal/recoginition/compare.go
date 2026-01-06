package recoginition

import (
	"image"
	"io"

	"github.com/rivo/duplo"
)

func CompareImages(srcpath, trgpath io.Reader) (duplo.Match, error) {

	store := duplo.New()

	srcImg, _, err := image.Decode(srcpath)

	if err != nil {
		return duplo.Match{}, err

	}
	trgImg, _, err := image.Decode(trgpath)

	if err != nil {
		return duplo.Match{}, err

	}

	srcHash, _ := duplo.CreateHash(srcImg)
	trgHash, _ := duplo.CreateHash(trgImg)

	store.Add("sourceImg", srcHash)
	matches := store.Query(trgHash)

	return *matches[0], nil

}
