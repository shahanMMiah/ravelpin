package recoginition

import (
	"github.com/galeone/tfgo"
	llama "github.com/ollama/ollama/api"
)

type ImageClassifier struct {
	Model          *tfgo.Model
	Path           string
	inputSigniture string
}

type LlamaResponseFuncs struct {
	ToolNameTable map[string]*LlamaResponseFunc
}

type LlamaResponseFunc struct {
	Tool         func(any) (string, error)
	OutputString string
	Err          error
}

type ClassifyModels struct {
	SearchClassify      *ImageClassifier
	YarnWeightClassify  *ImageClassifier
	LlammaClient        *llama.Client
	LlammaToolFuncs     llama.Tools
	LlammaStream        bool
	LlammaResponseTable LlamaResponseFuncs
}
