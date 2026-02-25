package recoginition

import (
	"github.com/sugarme/tokenizer/pretrained"
)

type TokenInput struct {
	Ids     []int32
	Masks   []int32
	TypeIds []int32
	Tokens  []string
}

func Tokenize(sentence string) (TokenInput, error) {
	// Download and cache pretrained tokenizer. In this case `bert-base-uncased` from Huggingface
	// can be any model with `tokenizer.json` available. E.g. `tiiuae/falcon-7b`
	/*
		configFile, err := tokenizer.CachedPath("bert-base-NER", "tokenizer.json")
		if err != nil {
			return TokenInput{}, err
		}
	*/

	tk, err := pretrained.FromFile("./model/bert_slim_ner/assets/tokenizer.json")
	if err != nil {
		return TokenInput{}, err

	}

	en, err := tk.EncodeSingle(sentence, true, true, true, true, true)
	if err != nil {
		return TokenInput{}, err
	}

	inputs := TokenInput{}

	for _, id := range en.GetIds() {
		inputs.Ids = append(inputs.Ids, int32(id))

	}
	for _, mask := range en.GetAttentionMask() {
		inputs.Masks = append(inputs.Masks, int32(mask))

	}
	for _, typ := range en.GetTypeIds() {
		inputs.TypeIds = append(inputs.TypeIds, int32(typ))

	}
	for _, toke := range en.GetTokens() {
		inputs.Tokens = append(inputs.Tokens, toke)

	}

	return inputs, nil
}
