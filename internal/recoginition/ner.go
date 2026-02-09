package recoginition

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tensorflow "github.com/galeone/tensorflow/tensorflow/go"
	"github.com/galeone/tfgo"
)

const NERMODELNAME = "bert_slim_ner"
const NERCONFIG = "config.json"

func GetLabelMap() (map[string]string, error) {
	config, err := os.ReadFile(fmt.Sprintf("./model/%s/%s", NERMODELNAME, NERCONFIG))
	labelMap := make(map[string]string)
	if err != nil {
		return labelMap, err
	}
	configData := map[string]any{}

	json.Unmarshal(config, &configData)

	if id, exist := configData["id2label"]; exist {
		if idMap, isMap := id.(map[string]any); isMap {
			for key, val := range idMap {
				vVal, _ := val.(string)
				labelMap[key] = vVal

			}
			return labelMap, nil
		}

	}

	return labelMap, fmt.Errorf("could not fine idlabel map")

}

func ArgMax(tens []*tensorflow.Tensor) ([]int32, error) {
	MaxPreds := make([]int32, 0)

	if result_tensor, isTensor := tens[0].Value().([][][]float32); isTensor {

		for ind, prdWeights := range result_tensor[0] {

			MaxPreds = append(MaxPreds, 0)
			var weightMaxInd int32 = 0
			var weightMax float32 = -1.0
			for maxInd, weight := range prdWeights {
				if weight > weightMax {
					weightMax = weight
					weightMaxInd = int32(maxInd)

				}
				MaxPreds[ind] = weightMaxInd
			}

		}
	}

	return MaxPreds, nil

}

func RealignTokens(predictions []int32, tokens []string, ids map[string]string) (map[string][]string, error) {

	ent := ""
	entities := []string{}
	outMap := make(map[string][]string)

	for num, token := range tokens {
		predIndex := predictions[num]
		predLabel := ids[strconv.Itoa(int(predIndex))]

		//fmt.Println(predLabel, token, predLabel)

		if strings.Contains(token, "##") {
			entities = append(entities, predLabel)
			ent += token[2:]
			continue

		}

		if predIndex != 0 && num != 0 {

			outMap[ent] = entities
			ent = ""
			entities = []string{}

			ent += token
			entities = append(entities, predLabel)
		}
	}

	return outMap, nil

}

func Ner(input string) (map[string][]string, error) {

	log.Printf("input is %v", input)

	modePath := fmt.Sprintf("./model/%s", NERMODELNAME)
	model := tfgo.LoadModel(modePath, []string{"serve"}, nil)

	tokenInput, err := Tokenize(input)
	if err != nil {
		return nil, err
	}

	wordTensor, err := tensorflow.NewTensor([][]int32{tokenInput.Ids})
	if err != nil {
		return nil, err
	}

	maskTensor, err := tensorflow.NewTensor([][]int32{tokenInput.Masks})
	if err != nil {
		return nil, err
	}
	typeTensor, err := tensorflow.NewTensor([][]int32{tokenInput.TypeIds})

	if err != nil {
		return nil, err
	}

	results := model.Exec(
		[]tensorflow.Output{
			model.Op("StatefulPartitionedCall", 0),
		},
		map[tensorflow.Output]*tensorflow.Tensor{

			model.Op("serving_default_token_type_ids", 0): typeTensor,
			model.Op("serving_default_attention_mask", 0): maskTensor,
			model.Op("serving_default_input_ids", 0):      wordTensor,
		},
	)
	predictions, err := ArgMax(results)
	if err != nil {
		return nil, err
	}

	//fmt.Printf("predictions -- %v", predictions)

	idLabels, err := GetLabelMap()
	if err != nil {
		return nil, err
	}

	//fmt.Println(idLabels)

	predictedLabels, err := RealignTokens(predictions, tokenInput.Tokens, idLabels)

	if err != nil {
		return nil, err
	}

	//fmt.Printf("predicted entities - %v", predictedLabels)

	return predictedLabels, nil
}
