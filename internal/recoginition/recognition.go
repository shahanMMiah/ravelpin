package recoginition

import (
	"bufio"
	"fmt"
	"image"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/disintegration/imaging"

	tensorflow "github.com/galeone/tensorflow/tensorflow/go"
	"github.com/galeone/tfgo"
)

const imgWH = 224

type byProbs []classification

func (a byProbs) Len() int           { return len(a) }
func (a byProbs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byProbs) Less(i, j int) bool { return a[i].Probability < a[j].Probability }

type classification struct {
	Label       string  `json:"label"`
	Probability float32 `json:"probability"`
}

type Labels []string

func GetImage(imagelink string) (image.Image, error) {
	response, err := http.Get(imagelink)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	img, err := imaging.Decode(response.Body)
	if err != nil {
		return nil, err
	}

	return img, nil

}

func ClasifyImageTest(imagelink string) []classification {
	os.Setenv("TF_CPP_MIN_LOG_LEVEL", "2")

	wd, _ := os.Getwd()

	model := tfgo.LoadModel(fmt.Sprintf("%s/model", wd), []string{"serve"}, nil)
	srcImage, err := GetImage(imagelink)
	if err != nil {
		log.Fatal(err)
	}
	scaledImg := imaging.Fill(srcImage, imgWH, imgWH, imaging.Center, imaging.Lanczos)

	imgTensor, _ := newImgTensor(imgWH, imgWH, scaledImg)

	results := model.Exec(
		[]tensorflow.Output{
			model.Op("StatefulPartitionedCall", 0),
		},
		map[tensorflow.Output]*tensorflow.Tensor{
			model.Op("serving_default_inputs", 0): imgTensor,
		},
	)

	labels, _ := loadLabels("./model")

	probabilities := results[0].Value().([][]float32)[0]

	classifications := []classification{}
	for i, p := range probabilities {
		if p < 5 {
			continue
		}
		classifications = append(classifications, classification{
			Label:       strings.ToLower(labels[i]),
			Probability: p,
		})
	}

	sort.Sort(byProbs(classifications))
	return classifications

}

func TestPrint() {
	fmt.Println("from recognition package")
}

func loadLabels(path string) ([]string, error) {
	labels := make([]string, 0)
	modelLabels := path + "/labels.txt"
	f, err := os.Open(modelLabels)

	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		labels = append(labels, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return labels, nil
}
func newImgTensor(imageHeight, imageWidth int, img *image.NRGBA) (*tensorflow.Tensor, error) {
	var tfImage [1][][][3]float32
	for j := 0; j < imageHeight; j++ {
		tfImage[0] = append(tfImage[0], make([][3]float32, imageWidth))
	}

	for i := 0; i < imageWidth; i++ {
		for j := 0; j < imageHeight; j++ {
			r, g, b, _ := img.At(i, j).RGBA()
			tfImage[0][j][i][0] = convertValue(r)
			tfImage[0][j][i][1] = convertValue(g)
			tfImage[0][j][i][2] = convertValue(b)
		}
	}

	return tensorflow.NewTensor(tfImage)

}

func convertValue(value uint32) float32 {
	return (float32(value >> 8)) / float32(255)
}
