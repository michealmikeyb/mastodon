package main

import (
	"log"
	"net/http"
	"os"
	"database/sql"
	"fmt"
	"strconv"
	"sort"

	_ "github.com/lib/pq"

	"github.com/gin-gonic/gin"
	"github.com/michealmikeyb/mastodon/sappho/fetchers"
	"github.com/michealmikeyb/mastodon/sappho/models"
	"github.com/michealmikeyb/mastodon/sappho/utils"
)

const (
	author_downrank_coefficient = 0.75
)
var	aggregate_weights = map[string]float32 {
	// Number of statuses by the author that the account liked
	"account_liked_author_status_count":	10.0,
	// Total number of statuses liked by the account
	"account_liked_status_count": 0.0,
	// Number of statuses that have a tag that the candidate has that the account liked
	"account_liked_tag_status_count": 10.0,
	// Number of statuses by the author that the account rebloged
	"account_rebloged_author_status_count": 30.0,
	// Total number of statuses the account rebloged
	"account_rebloged_status_count": 0,
	// Number of statuses that have a tag that the candidate has that the account rebloged
	"account_rebloged_tag_status_count": 20.0,
	// Number of followers the author has
	"author_follower_count": 0.004,
	// Number of likes on authors last 20 statuses
	"author_like_count": 0.4,
	// Number of reblogs on authors last 20 statuses
	"author_reblog_count": 0.8,
	// Number of replies on authors last 20 statuses
	"author_reply_count": 0.6,
	// Number of likes for candidate status
	"candidate_status_like_count": 1.0,
	// Number of reblogs for candidate status
	"candidate_status_reblog_count": 2.0,
	// Number of replies for candidate status
	"candidate_status_reply_count": 1.5,
	// average similarity between the candidate status open ai embedding 
	// and the average embedding for all the statuses liked by the account
	// will be in the 0 - 1000 range
	"average_like_embedding_similarity": 2,
	// average similarity between the candidate status open ai embedding 
	// and the average embedding for all the statuses rebloged by the account
	// will be in the 0 - 1000 range
	"average_reblog_embedding_similarity": 3,
	// Number of statuses liked by the account with a similar embedding
	"account_liked_status_with_similar_embedding": 8,
	// Number of statuses rebloged by the account with a similar embedding
	"account_rebloged_status_with_similar_embedding": 15,

}

type ByRank []models.RankedCandidate

func (a ByRank) Len() int           { return len(a) }
func (a ByRank) Less(i, j int) bool { return a[i].Rank < a[j].Rank }
func (a ByRank) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func author_downranker(candidates []models.RankedCandidate) []models.RankedCandidate {
	author_downranks := make(map[string]float32)
	sort.Sort(ByRank(candidates))
	downranked_candidates := make([]models.RankedCandidate, len(candidates))
	for i, candidate := range candidates {
		author_key := fmt.Sprintf("%s@%s", candidate.Candidate.AuthorUsername, candidate.Candidate.AuthorDomain)
		downrank, ok := author_downranks[author_key]
		downranked_candidate := models.RankedCandidate{
			Candidate: candidate.Candidate,
			Rank: candidate.Rank,
		}
		if ok {
			downranked_candidate.Rank = downranked_candidate.Rank * downrank
			author_downranks[author_key] = downrank * author_downrank_coefficient
		} else {
			author_downranks[author_key] = author_downrank_coefficient
		}
		downranked_candidates[i] = downranked_candidate
	}
	return downranked_candidates
}

// Get a postgres connection using environment variables
func get_postgres_conn() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	password := os.Getenv("DB_PASS")
	dbname := os.Getenv("DB_NAME")
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
    "password=%s dbname=%s sslmode=disable",
    host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	return db, nil
}


func main() {
	router := gin.Default()
	router.POST("/get_aggregates", getAggregatesHandler)
	router.POST("/get_rankings", getRankingsHandler)
	router.Run(":8080")
}


// takes in a list of candidates in json format and returns
// a list of aggregated candidates with the candidate and the aggregates as a json
func getAggregatesHandler(c *gin.Context) {
	db_conn, err := get_postgres_conn()
	if err != nil {
		c.IndentedJSON(http.StatusServiceUnavailable, map[string]string{"error": "error connecting to database"})
	}
	defer db_conn.Close()
	var candidates []models.Candidate
	err = c.BindJSON(&candidates)
	if err != nil {
		log.Panic(err)
	}
	res, err := fetchers.GetAggregates(candidates, db_conn)
	if err != nil {
		log.Panic(err)
	}
	err = utils.UpsertAggregates(db_conn, res)
	if err != nil {
		log.Panic(err)
	}
	c.IndentedJSON(http.StatusOK, res)
}

// takes a list of candidates in json format and returns
// a list of ranked candidates with the candidate and its ranking as json
func getRankingsHandler(c *gin.Context) {
	db_conn, err := get_postgres_conn()
	if err != nil {
		c.IndentedJSON(http.StatusServiceUnavailable, map[string]string{"error": "error connecting to database"})
	}
	defer db_conn.Close()
	var candidates []models.Candidate
	err = c.BindJSON(&candidates)
	if err != nil {
		log.Panic(err)
	}
	aggregated_candidates, err := fetchers.GetAggregates(candidates, db_conn)
	if err != nil {
		log.Panic(err)
	}
	ranked_candidates := make([]models.RankedCandidate, len(aggregated_candidates))
	for i, agg_cand := range aggregated_candidates {
		var rank float32
		rank = 0.0
		for agg_key, value := range aggregate_weights {
			weighted_aggregate := float32(agg_cand.Aggregates[agg_key]) * value
			rank = rank + weighted_aggregate
		}
		ranked_candidates[i] = models.RankedCandidate{
			Candidate: agg_cand.Candidate,
			Rank: rank,
		}
	}
	downranked_candidates := author_downranker(ranked_candidates)
	c.IndentedJSON(http.StatusOK, &downranked_candidates)
}
