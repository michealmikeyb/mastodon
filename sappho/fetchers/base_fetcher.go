package fetchers


import (
	"log"
	"github.com/michealmikeyb/mastodon/sappho/models"
	"database/sql"
)

type DataStreamMaps struct {
	Account 	AccountDataStreamMap
	Statuses 	StatusesDataStreamMap
}

func CreateDataStreamMaps(candidates []models.Candidate, db_conn *sql.DB) (DataStreamMaps, error) {
	data_stream_maps := DataStreamMaps{}
	account_dsm, err := GetAccountDataStreamMap(candidates, db_conn)
	if err != nil {
		return data_stream_maps, err
	}
	data_stream_maps.Account = account_dsm

	statuses_dsm, err := GetStatusesDataStreamMap(candidates, db_conn)
	if err != nil {
		return data_stream_maps, err
	}
	data_stream_maps.Statuses = statuses_dsm
	return data_stream_maps, nil
}

type Fetcher func(dsm DataStreamMaps, candidate models.Candidate, c chan int)

func FetchAuthorFollowersCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	author, err := dsm.Account["author_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting author follower count", err)
		close(c)
		return
	}
	c <- author.FollowersCount
	close(c)
	return
}

func FetchAuthorLikeCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["author_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting author follower count", err)
		close(c)
		return
	}
	like_count := 0
	for _, status := range *statuses {
		like_count = like_count + status.FavouritesCount
	}
	c <- like_count
	close(c)
	return
}

func FetchAccountLikedStatusCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["account_liked_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting account liked statuses", err)
		close(c)
		return
	}
	status_count := 0
	for range *statuses {
		status_count = status_count + 1
	}
	c <- status_count
	close(c)
	return
}

func FetchAccountReblogedStatusCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["account_rebloged_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting account rebloged statuses", err)
		close(c)
		return
	}
	status_count := 0
	for range *statuses {
		status_count = status_count + 1
	}
	c <- status_count
	close(c)
	return
}

func FetchAccountLikedAuthorStatusCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["account_liked_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting account liked statuses", err)
		close(c)
		return
	}
	status_count := 0
	for _, status := range *statuses {
		if (status.Account.Username == candidate.AuthorUsername && status.Account.Domain == candidate.AuthorDomain) {
			status_count = status_count + 1
		}
	}
	c <- status_count
	close(c)
	return
}
var AggregateFunctionMap = map[string]Fetcher{
	"author_follower_count": FetchAuthorFollowersCount,
	"author_like_count": FetchAuthorLikeCount,
	"account_liked_status_count": FetchAccountLikedStatusCount,
	"account_rebloged_status_count": FetchAccountReblogedStatusCount,
}


func GetAggregates(candidates []models.Candidate, db_conn *sql.DB) ([]models.AggregatedCandidate, error) {
	aggregated_candidates := make([]models.AggregatedCandidate, len(candidates))
	dsm, err := CreateDataStreamMaps(candidates, db_conn)
	if err != nil {
		return aggregated_candidates, err
	}
	chan_maps := make(map[string]map[models.Candidate] chan int)
	for aggregate, fetcher := range AggregateFunctionMap {
		chan_maps[aggregate] = make(map[models.Candidate] chan int)
		for _, candidate := range candidates {
			ch := make(chan int)
			chan_maps[aggregate][candidate] = ch
			go fetcher(dsm, candidate, ch)
		}
	}
	for i, candidate := range candidates {
		agg_candidate := models.AggregatedCandidate{
			Aggregates: make(map[string]int),
			Candidate: candidate,
		}
		for aggregate, _ := range AggregateFunctionMap {
			agg_candidate.Aggregates[aggregate] = <- chan_maps[aggregate][candidate]
		}
		aggregated_candidates[i] = agg_candidate
	}
	return aggregated_candidates, nil
}

