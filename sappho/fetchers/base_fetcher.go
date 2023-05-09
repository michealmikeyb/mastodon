/*
This is the core of the aggregate fetcher. Here the fetchers are defined which take
data from the data streams and aggregates them.
The aggregation is divided into two asynchronous parts:
	1. the data streams are initialized which will all run in parallel and fetch data
		for the fetchers to use.
	2. the fetchers will run, wait on the data they need from the data stream, and then
		use that data to generate aggregates on the candidate

each fetcher will create a given aggregate as defined by the AggregateFunctionMap
*/

package fetchers


import (
	"log"
	"github.com/michealmikeyb/mastodon/sappho/models"
	"github.com/michealmikeyb/mastodon/sappho/utils"
	"database/sql"
)

const (
	// embedding similarities will tend to cluster above this number
	embedding_similarity_adjustment = 6500
	// the threshold to check whether two embeddings are similar
	embedding_similarity_threshold = 7300
)

// Map for the different data stream types
type DataStreamMaps struct {
	Account 	AccountDataStreamMap
	Statuses 	StatusesDataStreamMap
	Status		StatusDataStreamMap
}

// Create each of the data streams and start them off
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

	status_dsm, err := GetStatusDataStreamMap(candidates, db_conn)
	if err != nil {
		return data_stream_maps, err
	}
	data_stream_maps.Status = status_dsm
	return data_stream_maps, nil
}

type Fetcher func(dsm DataStreamMaps, candidate models.Candidate, c chan int)

// Fetches the author follower count by getting the author stream and checking the 
// FollowersCount attribute
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

// Fetches the author like count by getting the author statuses stream and adding
// up the likes for the statuses
func FetchAuthorLikeCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["author_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting author like count", err)
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

// Fetches the author reblog count by getting the author statuses stream and adding
// up the reblogs for the statuses
func FetchAuthorReblogCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["author_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting author reblog count", err)
		close(c)
		return
	}
	reblog_count := 0
	for _, status := range *statuses {
		reblog_count = reblog_count + status.ReblogsCount
	}
	c <- reblog_count
	close(c)
	return
}

// Fetches the author reply count by getting the author statuses stream and adding
// up the replies for the statuses
func FetchAuthorReplyCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["author_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting author reply count", err)
		close(c)
		return
	}
	reply_count := 0
	for _, status := range *statuses {
		reply_count = reply_count + status.RepliesCount
	}
	c <- reply_count
	close(c)
	return
}

// Fetches the number of statuses liked by the candidate account by getting the account
// liked status stream and adding them up
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

// Fetches the number of statuses rebloged by the candidate account by getting the account
// rebloged status stream and adding them up
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

// Fetches the number of statuses the candidate account liked that were posted by the author
// by getting the account liked status stream and checking if the author username and domain match
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

// Fetches the number of statuses the candidate account rebloged that were posted by the author
// by getting the account rebloged status stream and checking if the author username and domain match
func FetchAccountReblogedAuthorStatusCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["account_rebloged_statuses_stream"].GetData(candidate)
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

// Fetches the number of statuses the candidate account liked that have a tag that the candidate status has
// by getting the account liked status stream and checking which statuses have a tag thats in the candidate status
func FetchAccountLikedTagStatusCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["account_liked_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting account liked statuses", err)
		close(c)
		return
	}
	candidate_status, err := dsm.Status["candidate_status"].GetData(candidate)
	if err != nil {
		log.Println("Error getting candidate status", err)
		close(c)
		return
	}

	status_count := 0
	for _, status := range *statuses {
		for _, candidate_tag := range candidate_status.Tags {
			for _, liked_tag := range status.Tags {
				if (candidate_tag.Name == liked_tag.Name) {
					status_count = status_count + 1
				}
			}
		}
	}
	c <- status_count
	close(c)
	return
}

// Fetches the number of statuses the candidate account rebloged that have a tag that the candidate status has
// by getting the account rebloged status stream and checking which statuses have a tag thats in the candidate status
func FetchAccountReblogedTagStatusCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["account_rebloged_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting account rebloged statuses", err)
		close(c)
		return
	}
	candidate_status, err := dsm.Status["candidate_status"].GetData(candidate)
	if err != nil {
		log.Println("Error getting candidate status", err)
		close(c)
		return
	}

	status_count := 0
	for _, status := range *statuses {
		for _, candidate_tag := range candidate_status.Tags {
			for _, liked_tag := range status.Tags {
				if (candidate_tag.Name == liked_tag.Name) {
					status_count = status_count + 1
				}
			}
		}
	}
	c <- status_count
	close(c)
	return
}

// Fetch the number of likes on the candidate status by getting the candidate status stream
// and checking the FavouritesCount attribute
func FetchCandidateStatusLikesCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	candidate_status, err := dsm.Status["candidate_status"].GetData(candidate)
	if err != nil {
		log.Println("Error getting candidate status", err)
		close(c)
		return
	}
	c <- candidate_status.FavouritesCount
	close(c)
	return
}

// Fetch the number of reblogs on the candidate status by getting the candidate status stream
// and checking the ReblogsCount attribute
func FetchCandidateStatusReblogCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	candidate_status, err := dsm.Status["candidate_status"].GetData(candidate)
	if err != nil {
		log.Println("Error getting candidate status", err)
		close(c)
		return
	}
	c <- candidate_status.ReblogsCount
	close(c)
	return
}

// Fetch the number of replies on the candidate status by getting the candidate status stream
// and checking the RepliesCount attribute
func FetchCandidateStatusReplyCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	candidate_status, err := dsm.Status["candidate_status"].GetData(candidate)
	if err != nil {
		log.Println("Error getting candidate status", err)
		close(c)
		return
	}
	c <- candidate_status.RepliesCount
	close(c)
	return
}

// Fetch the average embedding for the statuses the account likes and compare that 
// to the embedding of the candidate status
func FetchAverageLikeEmbeddingSimilarity(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["account_liked_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting account liked statuses", err)
		close(c)
		return
	}
	candidate_status, err := dsm.Status["candidate_status"].GetData(candidate)
	if err != nil {
		log.Println("Error getting candidate status", err)
		close(c)
		return
	}
	average_like_embedding := utils.GetEmbeddingAverage(*statuses)
	expanded_similarity := utils.GetAverageEmbeddingSimilarity(average_like_embedding, candidate_status.Embedding)
	if expanded_similarity < 1 {
		c <- 0
		close(c)
		return
	}
	// adjust since the similarities will cluster
	adjusted_similarity := expanded_similarity - embedding_similarity_adjustment
	c <- adjusted_similarity
	close(c)
	return
}

// Fetch the average embedding for the statuses the account likes and compare that 
// to the embedding of the candidate status
func FetchAverageReblogEmbeddingSimilarity(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["account_rebloged_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting account rebloged statuses", err)
		close(c)
		return
	}
	candidate_status, err := dsm.Status["candidate_status"].GetData(candidate)
	if err != nil {
		log.Println("Error getting candidate status", err)
		close(c)
		return
	}
	average_reblog_embedding := utils.GetEmbeddingAverage(*statuses)
	expanded_similarity := utils.GetAverageEmbeddingSimilarity(average_reblog_embedding, candidate_status.Embedding)
	if expanded_similarity < 1 {
		c <- 0
		close(c)
		return
	}
	c <- expanded_similarity - embedding_similarity_adjustment
	close(c)
	return
}

// Fetch the number of liked statuses that have a close embedding to
// the candidate status
func FetchLikedStatusesWithCloseEmbeddingCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["account_liked_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting account liked statuses", err)
		close(c)
		return
	}
	candidate_status, err := dsm.Status["candidate_status"].GetData(candidate)
	if err != nil {
		log.Println("Error getting candidate status", err)
		close(c)
		return
	}
	liked_status_count := 0
	for _, status := range *statuses {
		similarity := utils.GetAverageEmbeddingSimilarity(status.Embedding, candidate_status.Embedding)
		if similarity >= embedding_similarity_threshold {
			liked_status_count = liked_status_count + 1
		}
	}
	c <- liked_status_count
	close(c)
	return
	
}

// Fetch the number of rebloged statuses that have a close embedding to
// the candidate status
func FetchReblogedStatusesWithCloseEmbeddingCount(dsm DataStreamMaps, candidate models.Candidate, c chan int) {
	statuses, err := dsm.Statuses["account_rebloged_statuses_stream"].GetData(candidate)
	if err != nil {
		log.Println("Error getting account rebloged statuses", err)
		close(c)
		return
	}
	candidate_status, err := dsm.Status["candidate_status"].GetData(candidate)
	if err != nil {
		log.Println("Error getting candidate status", err)
		close(c)
		return
	}
	rebloged_status_count := 0
	for _, status := range *statuses {
		similarity := utils.GetAverageEmbeddingSimilarity(status.Embedding, candidate_status.Embedding)
		if similarity >= embedding_similarity_threshold {
			rebloged_status_count = rebloged_status_count + 1
		}
	}
	c <- rebloged_status_count
	close(c)
	return
	
}
// Maps the aggregate name to the function that is used to get it
var AggregateFunctionMap = map[string]Fetcher{
	"author_follower_count": FetchAuthorFollowersCount,
	"author_like_count": FetchAuthorLikeCount,
	"author_reblog_count": FetchAuthorReblogCount,
	"author_reply_count": FetchAuthorReplyCount,
	"account_liked_status_count": FetchAccountLikedStatusCount,
	"account_rebloged_status_count": FetchAccountReblogedStatusCount,
	"account_liked_author_status_count": FetchAccountLikedAuthorStatusCount,
	"account_rebloged_author_status_count": FetchAccountReblogedAuthorStatusCount,
	"account_liked_tag_status_count": FetchAccountLikedTagStatusCount,
	"account_rebloged_tag_status_count": FetchAccountReblogedTagStatusCount,
	"candidate_status_like_count": FetchCandidateStatusLikesCount,
	"candidate_status_reblog_count": FetchCandidateStatusReblogCount,
	"candidate_status_reply_count": FetchCandidateStatusReplyCount,
	"average_like_embedding_similarity": FetchAverageLikeEmbeddingSimilarity,
	"average_reblog_embedding_similarity": FetchAverageReblogEmbeddingSimilarity,
	"account_liked_status_with_similar_embedding": FetchLikedStatusesWithCloseEmbeddingCount,
	"account_rebloged_status_with_similar_embedding": FetchReblogedStatusesWithCloseEmbeddingCount,
}

// get the aggregates by initializing a data stream map and iterating through the 
// AggregateFunctionMap and running the function with a channel to return the aggregate
// to. It will then wait on all the channels and return the aggregates
func GetAggregates(candidates []models.Candidate, db_conn *sql.DB) ([]models.AggregatedCandidate, error) {
	aggregated_candidates := make([]models.AggregatedCandidate, len(candidates))
	dsm, err := CreateDataStreamMaps(candidates, db_conn)
	if err != nil {
		return aggregated_candidates, err
	}
	// used to hold the channels that the aggregates will be returned on
	// map looks like:
	// {
	//	aggregate_name: {
	//		candidate1: chan
	//		candidate2: chan 
	//	}	
	// }
	chan_maps := make(map[string]map[models.Candidate] chan int)
	for aggregate, fetcher := range AggregateFunctionMap {
		// create a chan map for each aggregate
		chan_maps[aggregate] = make(map[models.Candidate] chan int)
		for _, candidate := range candidates {
			// for each candidate create a channel for the aggregate to be returned on
			// once the fetcher is done
			ch := make(chan int)
			chan_maps[aggregate][candidate] = ch
			// start the fetcher asyncrhonously, will return the value on the channel
			go fetcher(dsm, candidate, ch)
		}
	}
	// iterate through the candidates and build an aggregate candidate
	for i, candidate := range candidates {
		agg_candidate := models.AggregatedCandidate{
			Aggregates: make(map[string]int),
			Candidate: candidate,
		}
		// iterate through the aggregates and wait for the fetcher to be done
		// and return its value on the channel
		for aggregate, _ := range AggregateFunctionMap {
			agg_candidate.Aggregates[aggregate] = <- chan_maps[aggregate][candidate]
		}
		aggregated_candidates[i] = agg_candidate
	}
	return aggregated_candidates, nil
}

