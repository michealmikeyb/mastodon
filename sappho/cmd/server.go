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
	router.GET("/get_aggregates", getAggregatesHandler)
	router.Run(":8080")
}



func getAggregatesHandler(c *gin.Context) {
	db_conn, err := get_postgres_conn()
	if err != nil {
		c.IndentedJSON(http.StatusServiceUnavailable, map[string]string{"error": "error connecting to database"})
	}
	defer db_conn.Close()
	candidate := models.Candidate{
		StatusExternalId: "110284314158405915",
		StatusDomain: "socel.net",
		StatusId: "110284314157790330",
		AuthorUsername: "BGP",
		AuthorDomain: "socel.net",
		AccountId: "110216720936469695",
	}
	candidates := make([]models.Candidate, 1)
	candidates[0] = candidate
	res, err := fetchers.GetAggregates(candidates, db_conn)
	if err != nil {
		log.Panic(err)
	}
	c.IndentedJSON(http.StatusOK, res)
}
