package models

type TrainingPoint struct {
	Aggregates				Aggregates `json:"aggregates"`
	Results					TrainingResults `json:"result"`
}

type TrainingResults struct {
	Liked			bool `json:"liked"`
	Rebloged		bool `json:"rebloged"`
}