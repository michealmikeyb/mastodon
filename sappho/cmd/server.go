package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/michealmikeyb/mastodon/sappho/fetchers"
)

func main() {
	router := gin.Default()
	router.GET("/", healthCheckHandler)
	router.Run(":8080")
}

func healthCheckHandler(c *gin.Context) {
	as := fetcher.AccountStream{}
	as.Init("https://sfba.social/api/v1/accounts/109277609058809814", "https://sfba.social/api/v1/accounts/109277609058809814")
	c.JSON(http.StatusOK, gin.H{
		"healthy": true,
		"secret":  "test",
	})
}
