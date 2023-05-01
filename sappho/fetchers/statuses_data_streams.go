package fetchers

import (
	"log"
	"net/http"
	"encoding/json"
	"github.com/michealmikeyb/mastodon/sappho/models"
	"fmt"
	"database/sql"
	"github.com/lib/pq"
)


type StatusesDataStream interface {
	Init(candidates []models.Candidate, db_conn *sql.DB) error
	GetData(candidate models.Candidate) (*[]models.Status, error)
	Close() error
}

type AuthorStatusesStream struct {
	candidates []models.Candidate
	channels map[string]chan []models.Status
	data map[string] []models.Status
}


func (as *AuthorStatusesStream) Init(candidates []models.Candidate, db_conn *sql.DB) error {
	as.candidates = candidates
	as.channels = make(map[string]chan []models.Status)
	as.data = make(map[string] []models.Status)
	for _, candidate := range candidates {
		author_key := fmt.Sprintf("%s@%s", candidate.AuthorUsername, candidate.AuthorDomain)
		if _, ok := as.channels[author_key] ; !ok {
			statuses_chan := get_author_statuses_channel(candidate)
			as.channels[author_key] = statuses_chan
		}
	}
	return nil
}

func get_author_statuses_channel(candidate models.Candidate) chan []models.Status{
	ch := make(chan []models.Status, 1)
	go func() {
		author_lookup_url := fmt.Sprintf("https://%s/api/v1/accounts/lookup?acct=%s", candidate.AuthorDomain, candidate.AuthorUsername)
		resp, err := http.Get(author_lookup_url)
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
		statuses_url := fmt.Sprintf("https://%s/api/v1/accounts/%s/statuses?exclude_replies=true", candidate.AuthorDomain, account.ID)
		resp, err = http.Get(statuses_url)
		if err != nil {
			log.Println("Error getting author statuses")
			close(ch)
			return
		}

		var statuses []models.Status
		err = json.NewDecoder(resp.Body).Decode(&statuses)
		if err != nil {
			log.Println("Error parsing author statuses ", err)
			close(ch)
			return
		}
		ch <- statuses
		return
	}()
	return ch
}

func (as *AuthorStatusesStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}

func (as *AuthorStatusesStream) GetData(candidate models.Candidate) (*[]models.Status, error) {
	author_key := fmt.Sprintf("%s@%s", candidate.AuthorUsername, candidate.AuthorDomain)
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusId, candidate.AccountUrl)
	}
	statuses, ok := as.data[author_key]
	if ok {
		return &statuses, nil
	}
	statuses_chan :=  as.channels[author_key]
	statuses = <-statuses_chan
	statuses_chan <- statuses
	as.data[author_key] = statuses
	return &statuses, nil

}
func (as *AuthorStatusesStream) Close() error {
	return nil
}

type AccountLikedStatusesStream struct {
	candidates []models.Candidate
	channels map[string]chan []models.Status
	data map[string] []models.Status
}


func (as *AccountLikedStatusesStream) Init(candidates []models.Candidate, db_conn *sql.DB) error {
	as.candidates = candidates
	as.channels = make(map[string]chan []models.Status)
	as.data = make(map[string] []models.Status)
	for _, candidate := range candidates {
		if _, ok := as.channels[candidate.AccountId] ; !ok {
			statuses_chan := get_account_liked_statuses_channel(candidate, db_conn)
			as.channels[candidate.AccountId] = statuses_chan
		}
	}
	return nil
}

func get_account_liked_statuses_channel(candidate models.Candidate, db_conn *sql.DB) chan []models.Status{
	ch := make(chan []models.Status, 1)
	go func() {
		rows, err := db_conn.Query(`SELECT statuses.id, statuses.created_at, statuses.in_reply_to_id, statuses.in_reply_to_account_id, 
		statuses.sensitive, statuses.spoiler_text, statuses.visibility, statuses.language, statuses.uri, 
		statuses.url, 
		COUNT(replies.id) AS replies_count, 
		COUNT(reblogs.id) AS reblogs_count,
		COUNT(status_favourites.id) AS favourites_count,
		statuses.edited_at, statuses.text,
		accounts.username, accounts.domain, 
		t.tag_array
		FROM favourites
		LEFT JOIN statuses ON favourites.status_id=statuses.id 
		LEFT JOIN statuses replies ON statuses.in_reply_to_id=replies.id 
		LEFT JOIN statuses reblogs ON statuses.reblog_of_id=reblogs.id 
		LEFT JOIN favourites status_favourites ON statuses.id=status_favourites.status_id 
		LEFT JOIN accounts ON statuses.account_id=accounts.id 
		LEFT OUTER JOIN (
			SELECT st.status_id AS status_id, array_agg(t.name) AS tag_array
			FROM   statuses_tags st
			JOIN   tags       t  ON t.id = st.tag_id
			GROUP  BY st.status_id
		)  t ON statuses.id = t.status_id
		WHERE favourites.account_id=$1
		GROUP BY statuses.id, t.tag_array, accounts.username, accounts.domain ;`, candidate.AccountId)
		if err != nil {
			log.Println("Error getting account liked statuses: ", err)
			close(ch)
			return
		}
		var statuses []models.Status
		for rows.Next() {
			status := models.Status{
				Account: models.Account{},
			}
			rows.Scan(&status.ID, &status.CreatedAt, &status.InReplyToID, &status.InReplyToAccountID, 
				&status.Sensitive, &status.SpoilerText, &status.Visibility, &status.Language, &status.URI, 
				&status.URL, &status.RepliesCount, &status.ReblogsCount, &status.FavouritesCount,
				&status.EditedAt, &status.Content, &status.Account.Username, &status.Account.Domain, pq.Array(&status.Tags),
			)
			statuses = append(statuses, status)
		}
		ch <- statuses
		return
	}()
	return ch
}

func (as *AccountLikedStatusesStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}

func (as *AccountLikedStatusesStream) GetData(candidate models.Candidate) (*[]models.Status, error) {
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusId, candidate.AccountUrl)
	}
	statuses, ok := as.data[candidate.AccountId]
	if ok {
		return &statuses, nil
	}
	statuses_chan :=  as.channels[candidate.AccountId]
	statuses = <-statuses_chan
	statuses_chan <- statuses
	as.data[candidate.AccountId] = statuses
	return &statuses, nil

}

func (as *AccountLikedStatusesStream)  Close() error {
	for _, ch := range as.channels {
		close(ch)
	}
	return nil
}

type AccountReblogedStatusesStream struct {
	candidates []models.Candidate
	channels map[string]chan []models.Status
	data map[string] []models.Status
}


func (as *AccountReblogedStatusesStream) Init(candidates []models.Candidate, db_conn *sql.DB) error {
	as.candidates = candidates
	as.channels = make(map[string]chan []models.Status)
	as.data = make(map[string] []models.Status)
	for _, candidate := range candidates {
		if _, ok := as.channels[candidate.AccountId] ; !ok {
			statuses_chan := get_account_rebloged_statuses_channel(candidate, db_conn)
			as.channels[candidate.AccountId] = statuses_chan
		}
	}
	return nil
}

func get_account_rebloged_statuses_channel(candidate models.Candidate, db_conn *sql.DB) chan []models.Status{
	ch := make(chan []models.Status, 1)
	go func() {
		rows, err := db_conn.Query(`SELECT statuses.id, statuses.created_at, statuses.in_reply_to_id, statuses.in_reply_to_account_id, 
		statuses.sensitive, statuses.spoiler_text, statuses.visibility, statuses.language, statuses.uri, 
		statuses.url, 
		COUNT(replies.id) AS replies_count, 
		COUNT(reblogs.id) AS reblogs_count,
		COUNT(status_favourites.id) AS favourites_count,
		statuses.edited_at, statuses.text, 
		t.tag_array,
		accounts.username, accounts.domain
		FROM statuses reblog
		INNER JOIN statuses ON reblog.reblog_of_id=statuses.id 
		LEFT JOIN statuses replies ON statuses.in_reply_to_id=replies.id 
		LEFT JOIN statuses reblogs ON statuses.reblog_of_id=reblogs.id 
		LEFT JOIN favourites status_favourites ON statuses.id=status_favourites.status_id 
		LEFT JOIN accounts ON statuses.account_id=accounts.id 
		LEFT OUTER JOIN (
			SELECT st.status_id AS status_id, array_agg(t.name) AS tag_array
			FROM   statuses_tags st
			JOIN   tags       t  ON t.id = st.tag_id
			GROUP  BY st.status_id
		)  t ON statuses.id = t.status_id
		WHERE reblog.account_id=$1
		GROUP BY statuses.id, t.tag_array, accounts.username, accounts.domain ;`, candidate.AccountId)
		if err != nil {
			log.Println("Error getting account liked statuses: ", err)
			close(ch)
			return
		}
		var statuses []models.Status
		for rows.Next() {
			status := models.Status{
				Account: models.Account{},
			}
			rows.Scan(&status.ID, &status.CreatedAt, &status.InReplyToID, &status.InReplyToAccountID, 
				&status.Sensitive, &status.SpoilerText, &status.Visibility, &status.Language, &status.URI, 
				&status.URL, &status.RepliesCount, &status.ReblogsCount, &status.FavouritesCount,
				&status.EditedAt, &status.Content, &status.Tags, &status.Account.Username, &status.Account.Domain, 
			)
			statuses = append(statuses, status)
		}
		ch <- statuses
		return
	}()
	return ch
}

func (as *AccountReblogedStatusesStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}

func (as *AccountReblogedStatusesStream) GetData(candidate models.Candidate) (*[]models.Status, error) {
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusId, candidate.AccountUrl)
	}
	statuses, ok := as.data[candidate.AccountId]
	if ok {
		return &statuses, nil
	}
	statuses_chan :=  as.channels[candidate.AccountId]
	statuses = <-statuses_chan
	statuses_chan <- statuses
	as.data[candidate.AccountId] = statuses
	return &statuses, nil

}

func (as *AccountReblogedStatusesStream) Close() error {
	return nil
}

type StatusesDataStreamMap map[string] StatusesDataStream

func GetStatusesDataStreamMap(candidates []models.Candidate, db_conn *sql.DB) (StatusesDataStreamMap, error) {
	data_stream_map := StatusesDataStreamMap{}
	data_stream_map["author_statuses_stream"] = &AuthorStatusesStream{}
	data_stream_map["account_liked_statuses_stream"] = &AccountLikedStatusesStream{}
	data_stream_map["account_rebloged_statuses_stream"] = &AccountReblogedStatusesStream{}

	for _, data_stream := range data_stream_map {
		err := data_stream.Init(candidates, db_conn)
		if err != nil {
			return data_stream_map, err
		}
	}
	return data_stream_map, nil

}

