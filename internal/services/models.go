package services

type RavelPhoto struct {
	MediumURL string `json:"medium_url"`
}

type YarnWeight struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type RavelryPattern struct {
	Id         int        `json:"id"`
	Name       string     `json:"name"`
	Permalink  string     `json:"permalink"`
	FirstPhoto RavelPhoto `json:"first_photo"`
}

type RavelryPatternFull struct {
	Id          int          `json:"id"`
	Name        string       `json:"name"`
	Permalink   string       `json:"permalink"`
	Photos      []RavelPhoto `json:"photos"`
	YarnWeights YarnWeight   `json:"yarn_weight"`
}
