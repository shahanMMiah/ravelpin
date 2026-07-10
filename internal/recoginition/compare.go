package recoginition

import (
	"fmt"
	"image"
	"io"
	"net/http"

	"golang.org/x/image/webp"

	"github.com/corona10/goimagehash"
	"github.com/disintegration/imaging"
	"github.com/rivo/duplo"
)

func GetImageBytes(imagelink string) ([]byte, error) {
	response, err := http.Get(imagelink)

	if err != nil {
		return nil, err
	}
	return io.ReadAll(response.Body)
}

func GetImage(imagelink string) (image.Image, error) {
	response, err := http.Get(imagelink)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	img, err := imaging.Decode(response.Body)

	if err != nil {

		_, contentType, err := image.DecodeConfig(response.Body)
		if err != nil {
			return nil, err
		}

		if contentType == "image/webp" {

			img, err := webp.Decode(response.Body)

			if err != nil {
				fmt.Printf("image format webp decode error %v", err)
				return nil, err
			}

			return img, nil
		}

		fmt.Printf("image format %v decode error %v", contentType, err)

		return nil, err
	}

	return img, nil

}

// percept hash

func HashFromInt(hashNum int64) *goimagehash.ImageHash {
	return goimagehash.NewImageHash(uint64(hashNum), goimagehash.PHash)
}

func CreatePHash(imgPath string) (*goimagehash.ImageHash, error) {
	img, err := GetImage(imgPath)

	if err != nil {
		return nil, err
	}

	return goimagehash.PerceptionHash(img)

}

func Split16BitHash(hash uint64) [4]int16 {

	return [4]int16{
		int16(hash >> 48),
		int16((hash >> 32) & 0xFFFF),
		int16((hash >> 16) & 0xFFFF),
		int16(hash & 0xFFFF),
	}
}

// duplo functions

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
