package fetchers

import (
	"log"
	"net/http"
	"encoding/json"
	"github.com/michealmikeyb/mastodon/sappho/models"
	"fmt"
	"database/sql"
	"github.com/lib/pq"
	"github.com/michealmikeyb/mastodon/sappho/utils"
)


// A data stream for an array of statuses
type StatusesDataStream interface {
	Init(candidates []models.Candidate, db_conn *sql.DB) error
	GetData(candidate models.Candidate) (*[]models.Status, error)
}

// A data stream for the statuses posted by the author
type AuthorStatusesStream struct {
	candidates []models.Candidate
	channels map[string]chan []models.Status
}

// initialize the author status stream with a set of candidates
func (as *AuthorStatusesStream) Init(candidates []models.Candidate, db_conn *sql.DB) error {
	as.candidates = candidates
	as.channels = make(map[string]chan []models.Status)
	for _, candidate := range candidates {
		// combine the author username and domain to get a unique author key
		author_key := fmt.Sprintf("%s@%s", candidate.AuthorUsername, candidate.AuthorDomain)
		if _, ok := as.channels[author_key] ; !ok {
			statuses_chan := get_author_statuses_channel(candidate)
			as.channels[author_key] = statuses_chan
		}
	}
	return nil
}

// Get a channel to send author status data to and start
// a function to fetch that data and send it through the channel
func get_author_statuses_channel(candidate models.Candidate) chan []models.Status{
	ch := make(chan []models.Status, 1)
	go func() {
		// first fetch the account by looking it up to get the account id
		author_lookup_url := fmt.Sprintf("https://%s/api/v1/accounts/lookup?acct=%s", candidate.AuthorDomain, candidate.AuthorUsername)
		resp, err := http.Get(author_lookup_url)
		if err != nil {
			log.Println("Error getting author", err)
			close(ch)
			return
		}

		var account models.Account
		err = json.NewDecoder(resp.Body).Decode(&account)
		if err != nil {
			log.Println("Error parsing author", err)
			close(ch)
			return
		}
		// use the account id to get the last 20 statuses from the author
		statuses_url := fmt.Sprintf("https://%s/api/v1/accounts/%s/statuses?exclude_replies=true", candidate.AuthorDomain, account.ID)
		resp, err = http.Get(statuses_url)
		if err != nil {
			log.Println("Error getting author statuses", err)
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

// used to confirm candidate present in list
func (as *AuthorStatusesStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}
// Get author status data from the channel and then push it back in
// in case another process needs it
func (as *AuthorStatusesStream) GetData(candidate models.Candidate) (*[]models.Status, error) {
	author_key := fmt.Sprintf("%s@%s", candidate.AuthorUsername, candidate.AuthorDomain)
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusId, candidate.AccountUrl)
	}
	statuses_chan :=  as.channels[author_key]
	statuses, ok := <-statuses_chan
	// if channel is closed just return an empty list
	if !ok {
		statuses = []models.Status{}
	// else send the statuses back in the channel in case another process needs them
	} else {
		statuses_chan <- statuses
	}
	return &statuses, nil

}

// A data stream for the statuses liked by the account
type AccountLikedStatusesStream struct {
	candidates []models.Candidate
	channels map[string]chan []models.Status
}

// initialize the account liked stream with a set of candidates
func (as *AccountLikedStatusesStream) Init(candidates []models.Candidate, db_conn *sql.DB) error {
	as.candidates = candidates
	as.channels = make(map[string]chan []models.Status)
	for _, candidate := range candidates {
		if _, ok := as.channels[candidate.AccountId] ; !ok {
			statuses_chan := get_account_liked_statuses_channel(candidate, db_conn)
			as.channels[candidate.AccountId] = statuses_chan
		}
	}
	return nil
}

// Get a channel to send account liked statuses data to and start
// a function to fetch that data and send it through the channel
func get_account_liked_statuses_channel(candidate models.Candidate, db_conn *sql.DB) chan []models.Status{
	ch := make(chan []models.Status, 1)
	go func() {
		// get the account liked statuses from the local db
		rows, err := db_conn.Query(`SELECT statuses.id, statuses.created_at, statuses.in_reply_to_id, statuses.in_reply_to_account_id, 
		statuses.sensitive, statuses.spoiler_text, statuses.visibility, statuses.language, statuses.uri, 
		statuses.url, 
		COUNT(replies.id) AS replies_count, 
		COUNT(reblogs.id) AS reblogs_count,
		COUNT(status_favourites.id) AS favourites_count,
		statuses.edited_at, statuses.text,
		accounts.username, accounts.domain, statuses.embedding,
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
		// after fetching scan the statuses into a status list
		for rows.Next() {
			var tag_strings []string
			status := models.Status{
				Account: models.Account{},
			}
			rows.Scan(&status.ID, &status.CreatedAt, &status.InReplyToID, &status.InReplyToAccountID, 
				&status.Sensitive, &status.SpoilerText, &status.Visibility, &status.Language, &status.URI, 
				&status.URL, &status.RepliesCount, &status.ReblogsCount, &status.FavouritesCount,
				&status.EditedAt, &status.Content, &status.Account.Username, &status.Account.Domain, 
				pq.Array(&status.Embedding), pq.Array(&tag_strings),
			)
			var tags []models.Tag
			for _, tag := range tag_strings {
				tags = append(tags, models.Tag{
					Name: tag,
				})
			}
			status.Tags = tags
			statuses = append(statuses, status)
		}
		referenced_statuses := make([]*models.Status, len(statuses))
		for i, _ := range statuses {
			referenced_statuses[i] = &statuses[i]
		}
		utils.GetStatusEmbeddingBulk(referenced_statuses, db_conn)
		for i, ref_status := range referenced_statuses {
			statuses[i] = *ref_status
		}
		ch <- statuses
		return
	}()
	return ch
}

// used to confirm candidate present in list
func (as *AccountLikedStatusesStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}

// Get account liked statuses data from the channel and then push it back in
// in case another process needs it
func (as *AccountLikedStatusesStream) GetData(candidate models.Candidate) (*[]models.Status, error) {
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusId, candidate.AccountUrl)
	}
	statuses_chan :=  as.channels[candidate.AccountId]
	statuses, ok := <-statuses_chan
	// if channel is closed just return an empty list
	if !ok {
		statuses = []models.Status{}
	// else send the statuses back in the channel in case another process needs them
	} else {
		statuses_chan <- statuses
	}
	return &statuses, nil

}

// A data stream for the statuses rebloged by the account
type AccountReblogedStatusesStream struct {
	candidates []models.Candidate
	channels map[string]chan []models.Status
}

// initialize the account rebloged stream with a set of candidates
func (as *AccountReblogedStatusesStream) Init(candidates []models.Candidate, db_conn *sql.DB) error {
	as.candidates = candidates
	as.channels = make(map[string]chan []models.Status)
	for _, candidate := range candidates {
		if _, ok := as.channels[candidate.AccountId] ; !ok {
			statuses_chan := get_account_rebloged_statuses_channel(candidate, db_conn)
			as.channels[candidate.AccountId] = statuses_chan
		}
	}
	return nil
}

// Get a channel to send account rebloged statuses data to and start
// a function to fetch that data and send it through the channel
func get_account_rebloged_statuses_channel(candidate models.Candidate, db_conn *sql.DB) chan []models.Status{
	ch := make(chan []models.Status, 1)
	go func() {
		// get the account rebloged statuses from the local db
		rows, err := db_conn.Query(`SELECT statuses.id, statuses.created_at, statuses.in_reply_to_id, statuses.in_reply_to_account_id, 
		statuses.sensitive, statuses.spoiler_text, statuses.visibility, statuses.language, statuses.uri, 
		statuses.url, 
		COUNT(replies.id) AS replies_count, 
		COUNT(reblogs.id) AS reblogs_count,
		COUNT(status_favourites.id) AS favourites_count,
		statuses.edited_at, statuses.text,
		accounts.username, accounts.domain, statuses.embedding,
		t.tag_array
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
		// after fetching scan the statuses into a status list
		for rows.Next() {
			var tag_strings []string
			status := models.Status{
				Account: models.Account{},
			}
			rows.Scan(&status.ID, &status.CreatedAt, &status.InReplyToID, &status.InReplyToAccountID, 
				&status.Sensitive, &status.SpoilerText, &status.Visibility, &status.Language, &status.URI, 
				&status.URL, &status.RepliesCount, &status.ReblogsCount, &status.FavouritesCount,
				&status.EditedAt, &status.Content, &status.Account.Username, &status.Account.Domain, 
				pq.Array(&status.Embedding), pq.Array(&tag_strings),
			)
			var tags []models.Tag
			for _, tag := range tag_strings {
				tags = append(tags, models.Tag{
					Name: tag,
				})
			}
			status.Tags = tags
			statuses = append(statuses, status)
		}
		referenced_statuses := make([]*models.Status, len(statuses))
		for i, _ := range statuses {
			referenced_statuses[i] = &statuses[i]
		}
		utils.GetStatusEmbeddingBulk(referenced_statuses, db_conn)
		for i, ref_status := range referenced_statuses {
			statuses[i] = *ref_status
		}
		ch <- statuses
		return
	}()
	return ch
}

// used to confirm candidate present in list
func (as *AccountReblogedStatusesStream) has_candidate(candidate models.Candidate) bool {
	for _, c := range as.candidates {
		if c == candidate {
			return true
		}
	}
	return false
}

// Get account rebloged statuses data from the channel and then push it back in
// in case another process needs it
func (as *AccountReblogedStatusesStream) GetData(candidate models.Candidate) (*[]models.Status, error) {
	if ! as.has_candidate(candidate) {
		return nil, fmt.Errorf("Candidate not in list with status url: %s and account url %s", candidate.StatusId, candidate.AccountUrl)
	}
	statuses_chan :=  as.channels[candidate.AccountId]
	statuses, ok := <-statuses_chan
	// if channel is closed just return an empty list
	if !ok {
		statuses = []models.Status{}
	// else send the statuses back in the channel in case another process needs them
	} else {
		statuses_chan <- statuses
	}
	return &statuses, nil

}

// A data stream map mapping the stream type to a datastream
type StatusesDataStreamMap map[string] StatusesDataStream

// Create an statuses datastream map and initialize all the datastreams
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
