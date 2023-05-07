package models


type OpenAiEmbeddingRequest struct {
	Input                 []string    `json:"input"`
	Model                 string    `json:"model"`
}

type OpenAiEmbeddingResponseData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type OpenAiEmbeddingResponseUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type OpenAiEmbeddingResponse struct {
	Object 			string 	`json:"object"`
	Data   			[]OpenAiEmbeddingResponseData  `json:"data"`
	Model 			string 	`json:"model"`
	Usage 			OpenAiEmbeddingResponseUsage `json:"usage"`
}