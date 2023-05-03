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

// A data stream for a singular account
type AccountDataStream interface {
	Init(candidates []models.Candidate, db_conn *sql.DB) error
	GetData(candidate models.Candidate) (*models.Account, error)
}

// A data stream for the author of the candidate status
type AuthorStream struct {
	candidates []models.Candidate
	channels map[string]chan models.Account
}

// initialize the author stream with a set of candidates
func (as *AuthorStream) Init(candidates []models.Candidate, db_conn *sql.DB) error {
	as.candidates = candidates
	as.channels = make(map[string]chan models.Account)
	for _, candidate := range candidates {
		// combine the author username and domain to get a unique author key
		author_key := fmt.Sprintf("%s@%s", candidate.AuthorUsername, candidate.AuthorDomain)
		if _, ok := as.channels[author_key] ; !ok {
			account_chan := get_author_channel(candidate)
			as.channels[author_key] = account_chan
		}
	}
	return nil
}

// Get a channel to send author data to and start
// a function to fetch that data and send it through the channel
func get_author_channel(candidate models.Candidate) chan models.Account{
	ch := make(chan models.Account, 1)
	go func() {
		// fetch the author information by looking up on the authors instance
		author_url := fmt.Sprintf("https://%s/api/v1/accounts/lookup?acct=%s", candidate.AuthorDomain, candidate.AuthorUsername)
		resp, err := http.Get(author_url)
		if err != nil {
			log.Println("Error getting author ", err)
			close(ch)
			return
		}

		var account models.Account
		err = json.NewDecoder(resp.Body).Decode(&account)
		if err != nil {
			log.Println("Error parsing author ", err)
			close(ch)
			return
		}
		ch <- account
		return
	}()
	return ch
}

// used to confirm candidate present in list
func (as *AuthorStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}

// Get author data from the channel and then push it back in
// in case another process needs it
func (as *AuthorStream) GetData(candidate models.Candidate) (*models.Account, error) {
	author_key := fmt.Sprintf("%s@%s", candidate.AuthorUsername, candidate.AuthorDomain)
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusId, candidate.AccountUrl)
	}
	account_chan :=  as.channels[author_key]
	account, ok  := <-account_chan
	// if channel is closed just send a blank account
	if !ok {
		account = models.Account{}
	// else send it back into the channel for another process to use
	} else {
		account_chan <- account
	}
	return &account, nil

}


// A data stream for the candidate account
type AccountStream struct {
	candidates []models.Candidate
	channels map[string]chan models.Account
}

// initialize the account stream with a set of candidates
func (as *AccountStream) Init(candidates []models.Candidate, db_conn *sql.DB) error {
	as.candidates = candidates
	as.channels = make(map[string]chan models.Account)
	for _, candidate := range candidates {
		if _, ok := as.channels[candidate.AccountUrl] ; !ok {
			account_chan := get_account_channel(candidate, db_conn)
			as.channels[candidate.AccountUrl] = account_chan
		}
	}
	return nil
}

// Make a channel to send the account data in and start a function to
// fetch the account data and send it to the channel
func get_account_channel(candidate models.Candidate, db_conn *sql.DB) chan models.Account{
	ch := make(chan models.Account, 1)
	// fetch the account from the local database
	go func() {
		account := models.Account{}
		err := db_conn.QueryRow(`SELECT username, display_name, locked, discoverable, note, 
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
		return
	}()
	return ch
}

// used to confirm candidate present in list
func (as *AccountStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}

// Get account data from the channel and then push it back in
// in case another process needs it
func (as *AccountStream) GetData(candidate models.Candidate) (*models.Account, error) {
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusId, candidate.AccountUrl)
	}
	account_chan :=  as.channels[candidate.AccountUrl]
	account, ok  := <-account_chan
	// if channel is closed just send a blank account
	if !ok {
		account = models.Account{}
	// else send it back into the channel for another process to use
	} else {
		account_chan <- account
	}
	return &account, nil

}

// A data stream map mapping the stream type to a datastream
type AccountDataStreamMap map[string] AccountDataStream

// Create an account datastream map and initialize all the datastreams
func GetAccountDataStreamMap(candidates []models.Candidate, db_conn *sql.DB) (AccountDataStreamMap, error) {
	data_stream_map := AccountDataStreamMap{}
	data_stream_map["account_stream"] = &AccountStream{}
	data_stream_map["author_stream"] = &AuthorStream{}

	for _, data_stream := range data_stream_map {
		err := data_stream.Init(candidates, db_conn)
		if err != nil {
			return data_stream_map, err
		}
	}
	return data_stream_map, nil

}