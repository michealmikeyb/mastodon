package fetchers

import (
	"log"
	"net/http"
	"encoding/json"
	"github.com/michealmikeyb/mastodon/sappho/models"
	"fmt"
	"database/sql"
)


type StatusDataStream interface {
	Init(candidates []models.Candidate, db_conn *sql.DB) error
	GetData(candidate models.Candidate) (*models.Status, error)
}

type CandidateStatusStream struct {
	candidates []models.Candidate
	channels map[string]chan models.Status
}


func (as *CandidateStatusStream) Init(candidates []models.Candidate, db_conn *sql.DB) error {
	as.candidates = candidates
	as.channels = make(map[string]chan models.Status)
	for _, candidate := range candidates {
		if _, ok := as.channels[candidate.StatusId] ; !ok {
			statuses_chan := get_candidate_channel(candidate)
			as.channels[candidate.StatusId] = statuses_chan
		}
	}
	return nil
}

func get_candidate_channel(candidate models.Candidate) chan models.Status{
	ch := make(chan models.Status, 1)
	go func() {
		status_url := fmt.Sprintf("https://%s/api/v1/statuses/%s", candidate.StatusDomain, candidate.StatusExternalId)
		resp, err := http.Get(status_url)
		if err != nil {
			log.Println("Error getting candidate status", err)
			close(ch)
			return
		}

		var status models.Status
		err = json.NewDecoder(resp.Body).Decode(&status)
		if err != nil {
			log.Println("Error parsing candidate status", err)
			close(ch)
			return
		}
		ch <- status
		return
	}()
	return ch
}

func (as *CandidateStatusStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}

func (as *CandidateStatusStream) GetData(candidate models.Candidate) (*models.Status, error) {
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusId, candidate.AccountUrl)
	}
	status_chan :=  as.channels[candidate.StatusId]
	status, ok := <-status_chan
	if !ok {
		status = models.Status{}
	} else {
		status_chan <- status
	}
	return &status, nil

}


type StatusDataStreamMap map[string] StatusDataStream

func GetStatusDataStreamMap(candidates []models.Candidate, db_conn *sql.DB) (StatusDataStreamMap, error) {
	data_stream_map := StatusDataStreamMap{}
	data_stream_map["candidate_status"] = &CandidateStatusStream{}

	for _, data_stream := range data_stream_map {
		err := data_stream.Init(candidates, db_conn)
		if err != nil {
			return data_stream_map, err
		}
	}
	return data_stream_map, nil

}

