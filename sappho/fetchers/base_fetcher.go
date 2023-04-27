package fetchers


import (
	"log"
	"github.com/michealmikeyb/mastodon/sappho/models"
)

type DataStreamMaps struct {
	Account AccountDataStreamMap
}

func CreateDataStreamMaps(candidates []models.Candidate) (DataStreamMaps, error) {
	data_stream_maps := DataStreamMaps{}
	account_dsm, err := GetAccountDataStreamMap(candidates)
	if err != nil {
		return data_stream_maps, err
	}
	data_stream_maps.Account = account_dsm
	return data_stream_maps, nil
}

type Fetcher func(dsm DataStreamMaps, candidate models.Candidate, c chan int)

func FetchAuthorFollowersCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	author, err := dsm.Account["author_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting author follower count")
		close(c)
		return
	}
	c <- author.FollowersCount
	close(c)
	return
}

func GetAggregates(candidates []models.Candidate) ([]models.AggregatedCandidate, error) {
	aggregated_candidates := make([]models.AggregatedCandidate, len(candidates))
	dsm, err := CreateDataStreamMaps(candidates)
	if err != nil {
		return aggregated_candidates, err
	}
	author_followers_count_chan_map := make(map[models.Candidate] chan int)
	for _, candidate := range candidates {
		ch := make(chan int)
		author_followers_count_chan_map[candidate] = ch
		go FetchAuthorFollowersCount(dsm, candidate, ch)
	}
	for i, candidate := range candidates {
		agg_candidate := models.AggregatedCandidate{}
		agg_candidate.Candidate = candidate
		agg_candidate.AuthorFollowerCount = <- author_followers_count_chan_map[candidate]
		aggregated_candidates[i] = agg_candidate
	}
	return aggregated_candidates, nil
}

