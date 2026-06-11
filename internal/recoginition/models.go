package recoginition

import "github.com/galeone/tfgo"

type ImageClassifier struct {
	Model          *tfgo.Model
	Path           string
	inputSigniture string
}

type ClassifyModels struct {
	SearchClassify     *ImageClassifier
	YarnWeightClassify *ImageClassifier
}
