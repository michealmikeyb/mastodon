package main

import (
	"log"
	"net/http"
	"os"
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/lib/pq"

	"github.com/gin-gonic/gin"
	"github.com/michealmikeyb/mastodon/sappho/fetchers"
	"github.com/michealmikeyb/mastodon/sappho/models"
)

var	aggregate_weights = map[string]float32 {
	"account_liked_author_status_count":	10.0,
	"account_liked_status_count": 0.0,
	"account_liked_tag_status_count": 10.0,
	"account_rebloged_author_status_count": 50.0,
	"account_rebloged_status_count": 0,
	"account_rebloged_tag_status_count": 40.0,
	"author_follower_count": 0.004,
	"author_like_count": 0.4,
	"author_reblog_count": 0.8,
	"author_reply_count": 0.6,
	"candidate_status_like_count": 1.0,
	"candidate_status_reblog_count": 2.0,
	"candidate_status_reply_count": 1.5,

}

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
	c.IndentedJSON(http.StatusOK, res)
}

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
	c.IndentedJSON(http.StatusOK, ranked_candidates)
}
