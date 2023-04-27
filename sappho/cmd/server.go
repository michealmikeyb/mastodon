package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/michealmikeyb/mastodon/sappho/fetchers"
	"github.com/michealmikeyb/mastodon/sappho/models"
)

func main() {
	router := gin.Default()
	router.GET("/get_aggregates", getAggregatesHandler)
	router.Run(":8080")
}



func getAggregatesHandler(c *gin.Context) {
	candidate := models.Candidate{
		AuthorUrl: "https://sfba.social/api/v1/accounts/109277609058809814",
		AccountId: "110216720936469695",
	}
	candidates := make([]models.Candidate, 1)
	candidates[0] = candidate
	res, err := fetchers.GetAggregates(candidates)
	if err != nil {
		log.Panic(err)
	}
	c.IndentedJSON(http.StatusOK, res)
}
