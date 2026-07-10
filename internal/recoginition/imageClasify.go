package recoginition

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/disintegration/imaging"

	tensorflow "github.com/galeone/tensorflow/tensorflow/go"
	"github.com/galeone/tfgo"

	llama "github.com/ollama/ollama/api"
)

const imgWH = 224
const CLASSIFYTHREASHOLD = 2

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

func (mdl *ImageClassifier) LoadModel(modelPath, inputSig string) {
	os.Setenv("TF_CPP_MIN_LOG_LEVEL", "2")
	mdl.Model = tfgo.LoadModel(modelPath, []string{"serve"}, nil)
	mdl.Path = modelPath
	mdl.inputSigniture = inputSig
}

func (model *ImageClassifier) GetClasifyLabels(Imagelink string, filterSclice []string) ([]string, error) {

	clsItems := make([]string, 0)

	// TEMP LABELS
	/*testRavelSearchTerms := map[string]any{
		"garment": []string{"sweater", "pancho", "cardigan", "trousers", "jean", "sock", "sweatshirt", "mitten"},
	}
	*/

	classified := model.ClassifyImage(Imagelink)

	if len(classified) < 1 {
		return nil, fmt.Errorf("couldnt classify whats in the image :(")
	}
	slog.Info(fmt.Sprintf("classified labels found %v\n", classified), slog.Any("labels", classified))

	for _, cls := range classified {

		if filterSclice == nil {
			clsItems = append(clsItems, cls.Label)
			continue
		}

		if slices.Contains(filterSclice, strings.ToLower(cls.Label)) {

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
			mdl.Model.Op(mdl.inputSigniture, 0): imgTensor,
		},
	)

	labels, _ := loadLabels(mdl.Path)

	probabilities := results[0].Value().([][]float32)[0]

	slog.Info(fmt.Sprintf("probalities for %s model: %v", mdl.Path, probabilities))
	classifications := []classification{}
	for i, p := range probabilities {
		if p < CLASSIFYTHREASHOLD {
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

// LAMA FUNCS

func NewTool(toolName, toolDescription, propertyName, propertyDescription, propertyType string) llama.Tool {

	propertyMap := llama.NewToolPropertiesMap()
	propertyMap.Set(propertyName, llama.ToolProperty{Type: llama.PropertyType{propertyType}, Description: propertyDescription})

	toolFunc := llama.ToolFunction{
		Name:        toolName,
		Description: toolDescription,
		Parameters: llama.ToolFunctionParameters{
			Type:       "object",
			Required:   []string{propertyName},
			Properties: propertyMap,
		},
	}
	return llama.Tool{
		Type:     "Function",
		Function: toolFunc,
	}

}

func NewYnTool() llama.Tool {
	return NewTool(
		os.Getenv("LLAMMAYNTOOLNAME"),
		"Guess Yarn Weight Name From Image",
		"name",
		"name of yarn weight",
		"string")
}

func NewClothingTool() llama.Tool {

	return NewTool(
		os.Getenv("LLAMMACLOTHTOOLNAME"),
		"Check The Input Clothing Name Is Valid",
		"name",
		"Name of Clothing Type",
		"string")

}

func NewResponseMap() LlamaResponseFuncs {

	return LlamaResponseFuncs{
		ToolNameTable: map[string]*LlamaResponseFunc{
			os.Getenv("LLAMMAYNTOOLNAME"):    &LlamaResponseFunc{Tool: GetYarnWeightName, OutputString: "", Err: nil},
			os.Getenv("LLAMMACLOTHTOOLNAME"): &LlamaResponseFunc{Tool: GetClothingName, OutputString: "", Err: nil},
		}}

}

func NewLlamaClient() (*llama.Client, error) {

	client, err := llama.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	initReq := &llama.GenerateRequest{
		Model: os.Getenv("LLAMAMODEL"),
		KeepAlive: &llama.Duration{
			Duration: -1, // Keep model loaded in RAM indefinitely
		},
	}

	slog.Info(fmt.Sprintf("init model test %v\n", initReq.Model))
	err = client.Generate(ctx, initReq, func(resp llama.GenerateResponse) error {
		return nil
	})

	if err != nil {
		return nil, err
	}

	return client, nil

}

func CheckValidGuess(name any, names map[string]interface{}) (string, error) {
	nme, valid := name.(string)
	if valid {
		if _, exists := names[strings.ToLower(nme)]; exists {
			return nme, nil
		}
	}
	return "", fmt.Errorf("Guess %s is not Valid", name)
}

func GetYarnWeightName(name any) (string, error) {
	yns := map[string]interface{}{
		"lace":      nil,
		"fingering": nil,
		"sport":     nil,
		"dk":        nil,
		"aran":      nil,
		"bulky":     nil,
		"jumbo":     nil,
	}

	return CheckValidGuess(name, yns)
}

func GetClothingName(name any) (string, error) {
	clth := map[string]interface{}{
		"coat":             nil,
		"dress":            nil,
		"intimate-apparel": nil,
		"leggings":         nil,
		"onesies":          nil,
		"pants":            nil,
		"robe":             nil,
		"shorts":           nil,
		"shrug":            nil,
		"skirt":            nil,
		"sleepwear":        nil,
		"soakers":          nil,
		"sweater":          nil,
		"swimwear":         nil,
		"tops":             nil,
		"vest":             nil,
		"other-clothing":   nil,
	}
	return CheckValidGuess(name, clth)
}

func (rMap *LlamaResponseFuncs) SetError(toolName string, err error) {
	rMap.ToolNameTable[toolName].Err = err
}

func (rMap *LlamaResponseFuncs) SetOutputString(toolName, val string) {
	rMap.ToolNameTable[toolName].OutputString = val
}

func (rMap *LlamaResponseFuncs) GetOutputString(toolName string) string {
	return rMap.ToolNameTable[toolName].OutputString

}

func (rMap *LlamaResponseFuncs) CheckForErr() error {
	for _, tool := range rMap.ToolNameTable {
		if tool.Err != nil {
			return tool.Err
		}
	}
	return nil
}

func NewResponseFunc(responseMap *LlamaResponseFuncs) func(resp llama.ChatResponse) error {

	return func(resp llama.ChatResponse) error {

		fmt.Println(resp.Message)
		if len(resp.Message.ToolCalls) == 0 {
			return fmt.Errorf("no tool funcs called.")
		}

		for _, fnc := range resp.Message.ToolCalls {
			toolFunc, valid := responseMap.ToolNameTable[fnc.Function.Name]
			if !valid {
				return fmt.Errorf("tool %s not found %v.", toolFunc, responseMap.ToolNameTable)
			}

			argm, valid := fnc.Function.Arguments.Get("name")
			if !valid {
				return fmt.Errorf("tool param  %s not found %v.", argm, responseMap.ToolNameTable)

			}

			nme, err := toolFunc.Tool(argm)
			if err != nil {
				return err
			}
			responseMap.SetOutputString(fnc.Function.Name, nme)

		}

		return nil

	}

}

func (cls *ClassifyModels) ClassifyImageYn(img []byte) error {
	msg := []llama.Message{
		{
			Role:     "user",
			Thinking: "low",
			Content:  os.Getenv("YNTTOOLPROMPT"),
			Images:   []llama.ImageData{img},
		},
	}

	req := &llama.ChatRequest{
		Model:    os.Getenv("LLAMAMODEL"),
		Messages: msg,
		Tools:    llama.Tools{cls.LlammaToolFuncs[0]},
		Stream:   &cls.LlammaStream,
	}

	respFunc := NewResponseFunc(&cls.LlammaResponseTable)

	err := cls.LlammaClient.Chat(context.Background(), req, respFunc)
	if err != nil {
		cls.LlammaResponseTable.SetError(os.Getenv("LLAMMAYNTOOLNAME"), err)
	}

	return nil

}

func (cls *ClassifyModels) ClassifyImageCloth(img []byte) {
	msg := []llama.Message{
		{
			Role:     "user",
			Thinking: "low",
			Content:  os.Getenv("CLOTHTOOLPROMPT"),
			Images:   []llama.ImageData{img},
		},
	}

	req := &llama.ChatRequest{
		Model:    os.Getenv("LLAMAMODEL"),
		Messages: msg,
		Tools:    llama.Tools{cls.LlammaToolFuncs[1]},
		Stream:   &cls.LlammaStream,
	}

	respFunc := NewResponseFunc(&cls.LlammaResponseTable)

	err := cls.LlammaClient.Chat(context.Background(), req, respFunc)
	if err != nil {
		cls.LlammaResponseTable.SetError(os.Getenv("LLAMMACLOTHTOOLNAME"), err)
	}

}

func (cls *ClassifyModels) ClassifyImageDetails(img []byte) error {

	wg := sync.WaitGroup{}

	wg.Go(func() { cls.ClassifyImageYn(img) })
	wg.Go(func() { cls.ClassifyImageCloth(img) })

	wg.Wait()

	err := cls.LlammaResponseTable.CheckForErr()
	if err != nil {
		return err
	}

	return nil

}

// LEGACY TEST TO REMOVE

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
