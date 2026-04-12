package recoginition

import (
	"bufio"
	"fmt"
	"image"
	"log/slog"
	"os"
	"slices"
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

func NewModel() *ImageClassifier {

	return new(ImageClassifier)
}

func (mdl *ImageClassifier) LoadModel(modelPath string) {
	os.Setenv("TF_CPP_MIN_LOG_LEVEL", "2")
	mdl.Model = tfgo.LoadModel(modelPath, []string{"serve"}, nil)
	mdl.Path = modelPath

}

func (model *ImageClassifier) GetClasifyLabels(Imagelink string) ([]string, error) {

	clsItems := make([]string, 0)

	// TEMP LABELS
	testRavelSearchTerms := map[string]any{
		"garment": []string{"sweater", "pancho", "cardigan", "trousers", "jean", "sock", "sweatshirt", "mitten"},
	}

	classified := model.ClassifyImage(Imagelink)

	if len(classified) < 1 {
		return nil, fmt.Errorf("couldnt classify whats in the image :(")
	}
	slog.Info(fmt.Sprintf("classified labels found %v\n", classified), slog.Any("labels", classified))

	for _, cls := range classified {

		garmList, _ := testRavelSearchTerms["garment"].([]string)

		if slices.Contains(garmList, strings.ToLower(cls.Label)) {

			clsItems = append(clsItems, cls.Label)

		}
	}

	return clsItems, nil
}

func (mdl *ImageClassifier) ClassifyImage(imagelink string) []classification {

	srcImage, err := GetImage(imagelink)
	if err != nil {
		slog.Error(err.Error())
	}
	scaledImg := imaging.Fill(srcImage, imgWH, imgWH, imaging.Center, imaging.Lanczos)

	imgTensor, _ := newImgTensor(imgWH, imgWH, scaledImg)

	results := mdl.Model.Exec(
		[]tensorflow.Output{
			mdl.Model.Op("StatefulPartitionedCall", 0),
		},
		map[tensorflow.Output]*tensorflow.Tensor{
			mdl.Model.Op("serving_default_inputs", 0): imgTensor,
		},
	)

	labels, _ := loadLabels(mdl.Path)

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

/* LEGACY TEST TO REMOVE*/

func ClasifyImageTest(imagelink string) []classification {
	os.Setenv("TF_CPP_MIN_LOG_LEVEL", "2")

	modelPath := fmt.Sprintf("./model/%s", "mobilenet")
	model := tfgo.LoadModel(modelPath, []string{"serve"}, nil)
	srcImage, err := GetImage(imagelink)
	if err != nil {
		slog.Error(err.Error())
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

	labels, _ := loadLabels(modelPath)

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
