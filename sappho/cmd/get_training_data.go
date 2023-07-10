package main

import (
	"strconv"
	"database/sql"
	"os"
	"encoding/json"
	"io/ioutil"
	"fmt"
	"log"
	
	_ "github.com/lib/pq"
	"github.com/michealmikeyb/mastodon/sappho/models"
)

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

// gets all the statuses that were liked by a local user
func get_training_data(db_conn *sql.DB) []models.TrainingPoint {
	rows, err := db_conn.Query(`SELECT aggregates.aggregate, 
		favourites.id IS NOT NULL as liked,
		reblog.id IS NOT NULL as rebloged
		FROM aggregates 
		LEFT JOIN favourites ON favourites.status_id=aggregates.status_id AND favourites.account_id=aggregates.account_id
		LEFT JOIN statuses reblog ON reblog.reblog_of_id=aggregates.status_id AND reblog.account_id=aggregates.account_id
		WHERE aggregates.seen;`)
	if err != nil {
		panic(err)
	}
	var training_points []models.TrainingPoint
	for rows.Next() {
		training_point := models.TrainingPoint{}
		rows.Scan(&training_point.Aggregates, &training_point.Results.Liked, &training_point.Results.Rebloged)
		training_points = append(training_points, training_point)
	}
	return training_points
}

// gets embedding for previous likes and reblogs
func main() {
	db_conn, err := get_postgres_conn()
	if err != nil {
		log.Println("Error connecting to db ")
		return
	}
	training_points := get_training_data(db_conn)
	file, _ := json.MarshalIndent(training_points, "", " ")
	_ = ioutil.WriteFile("training_data.json", file, 0644)
}
