package fetchers

import (
	"log"
	"net/http"
	"encoding/json"
	"github.com/michealmikeyb/mastodon/sappho/models"
	"fmt"
	"database/sql"

	_ "github.com/lib/pq"
)


const (
	host     = "127.0.0.1"
	port     = 5432
	user     = "postgres"
	password = ""
	dbname   = "mastodon"
  )

type AccountDataStream interface {
	Init(candidates []models.Candidate) error
	GetData(candidate models.Candidate) (*models.Account, error)
}

type AuthorStream struct {
	candidates []models.Candidate
	channels map[string]chan models.Account
	data map[string] models.Account
}


func (as *AuthorStream) Init(candidates []models.Candidate) error {
	as.candidates = candidates
	as.channels = make(map[string]chan models.Account)
	as.data = make(map[string] models.Account)
	for _, candidate := range candidates {
		if _, ok := as.channels[candidate.AuthorUrl] ; !ok {
			account_chan := get_author_channel(candidate)
			as.channels[candidate.AuthorUrl] = account_chan
		}
	}
	return nil
}

func get_author_channel(candidate models.Candidate) chan models.Account{
	ch := make(chan models.Account)
	go func() {
		resp, err := http.Get(candidate.AuthorUrl)
		if err != nil {
			log.Println("Error getting author")
			close(ch)
			return
		}

		var account models.Account
		err = json.NewDecoder(resp.Body).Decode(&account)
		if err != nil {
			log.Println("Error parsing author")
			close(ch)
			return
		}
		ch <- account
		close(ch)
		return
	}()
	return ch
}

func (as *AuthorStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}

func (as *AuthorStream) GetData(candidate models.Candidate) (*models.Account, error) {
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusUrl, candidate.AccountUrl)
	}
	account, ok := as.data[candidate.AuthorUrl]
	if ok {
		return &account, nil
	}
	account_chan :=  as.channels[candidate.AuthorUrl]
	account = <-account_chan
	as.data[candidate.AuthorUrl] = account
	return &account, nil

}



type AccountStream struct {
	candidates []models.Candidate
	channels map[string]chan models.Account
	data map[string] models.Account
}

func get_postgres_conn() (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
    "password=%s dbname=%s sslmode=disable",
    host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	return db, nil
}


func (as *AccountStream) Init(candidates []models.Candidate) error {
	as.candidates = candidates
	as.channels = make(map[string]chan models.Account)
	as.data = make(map[string] models.Account)
	for _, candidate := range candidates {
		if _, ok := as.channels[candidate.AuthorUrl] ; !ok {
			account_chan := get_account_channel(candidate)
			as.channels[candidate.AccountUrl] = account_chan
		}
	}
	return nil
}

func get_account_channel(candidate models.Candidate) chan models.Account{
	ch := make(chan models.Account)
	go func() {
		db, err := get_postgres_conn()
		if err != nil {
			log.Println("Error connecting to the db")
			close(ch)
			return
		}
		account := models.Account{}
		err = db.QueryRow(`SELECT username, display_name, locked, discoverable, note, 
			(SELECT count(*) FROM follows WHERE follows.account_id = accounts.id) AS following_count, 
			(SELECT count(*) FROM follows WHERE follows.target_account_id = accounts.id) AS follower_count, 
			(SELECT count(*) FROM statuses WHERE statuses.account_id = accounts.id) AS statuses_count  
			FROM accounts WHERE accounts.id = $1`, candidate.AccountId).Scan(&account.Username, &account.DisplayName, &account.Locked, &account.Discoverable, &account.Note, &account.FollowingCount, &account.FollowersCount, &account.StatusesCount)
		if err != nil {
			log.Println("Error receiving records from db %s", err)
			close(ch)
			return
		}
		ch <- account
		close(ch)
		return
	}()
	return ch
}


func (as *AccountStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}

func (as *AccountStream) GetData(candidate models.Candidate) (*models.Account, error) {
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusUrl, candidate.AccountUrl)
	}
	account, ok := as.data[candidate.AccountUrl]
	if ok {
		return &account, nil
	}
	account_chan :=  as.channels[candidate.AccountUrl]
	account = <-account_chan
	as.data[candidate.AccountUrl] = account
	return &account, nil

}

type AccountDataStreamMap map[string] AccountDataStream

func GetAccountDataStreamMap(candidates []models.Candidate) (AccountDataStreamMap, error) {
	data_stream_map := AccountDataStreamMap{}
	data_stream_map["account_stream"] = &AccountStream{}
	data_stream_map["author_stream"] = &AuthorStream{}

	for _, data_stream := range data_stream_map {
		err := data_stream.Init(candidates)
		if err != nil {
			return data_stream_map, err
		}
	}
	return data_stream_map, nil

}