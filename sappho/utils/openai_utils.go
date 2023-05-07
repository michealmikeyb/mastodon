package utils

import (
	"log"
	"net/http"
	"encoding/json"
	"github.com/michealmikeyb/mastodon/sappho/models"
	"fmt"
	"bytes"
	"strconv"
	"os"
	"math"
	"database/sql"
	"github.com/lib/pq"
)

// Get a statuses embedding, first checking if its already present on the status
// then checking if its already in the db, then getting it directly from open ai
// and saving it to the db
func GetStatusEmbedding(status *models.Status, db_conn *sql.DB) {
	// if its already on the status return
	if len(status.Embedding) > 0 {
		return
	}

	// next check to see if its on the db
	var embedding []float64
	err := db_conn.QueryRow(`SELECT embedding FROM statuses WHERE statuses.id = $1`, status.ID).Scan(pq.Array(&embedding))
	if (err == nil && len(embedding) > 0) {
		status.Embedding = embedding
		return
	}

	// next get it from openai
	openai_key := os.Getenv("OPENAI_KEY")
	api_url := "https://api.openai.com/v1/embeddings"
	open_ai_req := models.OpenAiEmbeddingRequest{
		Input: []string{status.Content},
		Model: "text-embedding-ada-002",
	}
	marshalled_req, err := json.Marshal(open_ai_req)
	if err != nil {
		log.Println("Error marshalling status to get embedding ", err)
		return
	}
	request, err := http.NewRequest("POST", api_url, bytes.NewReader(marshalled_req))
	if err != nil {
		log.Println("Error making request to openai ", err)
		return
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", openai_key))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Println("Error getting embedding ", err)
		return
	}
	defer resp.Body.Close()

	var openai_res models.OpenAiEmbeddingResponse
	err = json.NewDecoder(resp.Body).Decode(&openai_res)
	if err != nil {
		log.Println("Error parsing open ai response ", err)
		return
	}
	if len(openai_res.Data) != 0 {
		embedding = openai_res.Data[0].Embedding
		status.Embedding = embedding
		// save it to the db
		_, err = db_conn.Exec("UPDATE statuses SET embedding = $1 WHERE statuses.id = $2", pq.Array(embedding), status.ID)
		if err != nil {
			log.Println("Error puting embedding in db ", err)
			return
		}
	}
}

// Get a set of status embedding, first checking if they are already present on the status
// then checking if they are already in the db, then getting them directly from open ai
// and saving them to the db
func GetStatusEmbeddingBulk(statuses []*models.Status, db_conn *sql.DB) {
	// get all the statuses without an embedding on them
	var statuses_without_embedding []*models.Status
	for _, status := range statuses {
		if len(status.Embedding) > 0 {
			continue
		} else {
			statuses_without_embedding = append(statuses_without_embedding, status)
		}

	}
	statuses_without_embedding_ids := make([]int, len(statuses_without_embedding))
	// check if the status has an embedding in the db
	for _, status := range statuses_without_embedding {
		id, err := strconv.Atoi(status.ID)
		if err != nil {
			log.Println("Error parsing status id ", err)
		} else {
			statuses_without_embedding_ids = append(statuses_without_embedding_ids, id)
		}
	}
	rows, err := db_conn.Query(`SELECT embedding, id FROM statuses WHERE statuses.id = ANY($1)`, pq.Array(statuses_without_embedding_ids))
	if err != nil {
		log.Println("Error getting embeddings from db ", err)
		return
	}
	for rows.Next() {
		var status_id string
		var embedding []float64
		rows.Scan(pq.Array(&embedding), &status_id)
		for _, status := range statuses_without_embedding {
			if status_id == status.ID {
				status.Embedding = embedding
			}
		}
	}
	// get the embedding from openai
	var statuses_without_db_embedding []*models.Status
	for _, status := range statuses {
		if len(status.Embedding) > 0 {
			continue
		} else {
			statuses_without_db_embedding = append(statuses_without_embedding, status)
		}

	}
	var statuses_without_db_embedding_content []string
	for _, status := range statuses_without_db_embedding {
		statuses_without_db_embedding_content = append(statuses_without_db_embedding_content, status.Content)
	}
	openai_key := os.Getenv("OPENAI_KEY")
	api_url := "https://api.openai.com/v1/embeddings"
	open_ai_req := models.OpenAiEmbeddingRequest{
		Input: statuses_without_db_embedding_content,
		Model: "text-embedding-ada-002",
	}
	marshalled_req, err := json.Marshal(open_ai_req)
	if err != nil {
		log.Println("Error marshalling statuses to get embedding ", err)
		return
	}
	request, err := http.NewRequest("POST", api_url, bytes.NewReader(marshalled_req))
	if err != nil {
		log.Println("Error making request to openai ", err)
		return
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", openai_key))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Println("Error getting embedding ", err)
		return
	}
	defer resp.Body.Close()

	var openai_res models.OpenAiEmbeddingResponse
	err = json.NewDecoder(resp.Body).Decode(&openai_res)
	if err != nil {
		log.Println("Error parsing open ai response ", err)
		return
	}
	log.Println("got this many embeddings from open ai ", len(openai_res.Data))
	for i, res_data := range openai_res.Data {
		embedding := res_data.Embedding
		statuses_without_db_embedding[i].Embedding = embedding
		log.Println("putting in db")
		_, err = db_conn.Exec("UPDATE statuses SET embedding = $1 WHERE statuses.id = $2", pq.Array(embedding), statuses_without_db_embedding[i].ID)
		if err != nil {
			log.Println("Error puting embedding in db ", err)
			return
		}
	}
}

type EmbeddingAverage struct {
	Average		float64
	Index		int
}

// get the average for each of the variables in the embeddings for a set of statuses
func GetEmbeddingAverage(statuses []models.Status) []float64 {
	cleaned_statuses := []models.Status{}
	for _, status := range statuses {
		if len(status.Embedding) == 0 {
			continue
		}
		cleaned_statuses = append(cleaned_statuses, status)
	}
	if len(cleaned_statuses) == 0 {
		log.Println("No statuses gotten returning 0 array")
		return make([]float64, 1536)
	}
	average_chan := make(chan EmbeddingAverage)
	for i, _ := range cleaned_statuses[0].Embedding {
		go func() {
			var sum float64
			sum = 0.0
			for _, status := range cleaned_statuses {
				sum = sum + status.Embedding[i]
			}
			average := sum / float64(len(cleaned_statuses))
			average_chan <- EmbeddingAverage{
				Average: average,
				Index: i,
			}
		}()
	}

	averages := make([]float64, len(cleaned_statuses[0].Embedding))
	averages_received := 0
	for embedding_average := range average_chan {
		averages[embedding_average.Index] = embedding_average.Average
		averages_received = averages_received +1
		if averages_received >= len(cleaned_statuses[0].Embedding) {
			break
		}
	}
	return averages
}

func GetAverageEmbeddingDifference(embedding1 []float64, embedding2 []float64) float64 {
	if len(embedding1) != len(embedding2) {
		log.Println("Missing embedding ")
		return 1.0
	}
	var sum float64
	sum = 0.0
	for i, _ := range embedding1 {
		sum = sum + math.Abs(embedding1[i] - embedding2[i])
	}
	average := sum / float64(len(embedding1))
	return average
}