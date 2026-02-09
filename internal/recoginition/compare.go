package recoginition

import (
	"github.com/rivo/duplo"
)

func CompareImage(srcpath, trgpath string) (duplo.Match, error) {

	store := CreateStore()
	trgHash, err := CreateHash(trgpath)
	if err != nil {
		return duplo.Match{}, err

	}
	AddToStore(store, "src", srcpath)
	matches := store.Query(trgHash)

	return *matches[0], nil

}

func CreateStore() *duplo.Store {
	return duplo.New()
}

func CreateHash(imgPath string) (duplo.Hash, error) {
	img, err := GetImage(imgPath)

	if err != nil {
		return duplo.Hash{}, err

	}

	imgHash, _ := duplo.CreateHash(img)
	return imgHash, nil

}
func AddToStore(store *duplo.Store, hshName any, imgpath string) error {

	hsh, err := CreateHash(imgpath)

	if err != nil {
		return err
	}

	store.Add(hshName, hsh)

	return nil

}
