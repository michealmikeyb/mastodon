package main

import (
	"log"
	"fmt"
	"strconv"
	"database/sql"
	"os"

	_ "github.com/lib/pq"
	"github.com/michealmikeyb/mastodon/sappho/utils"
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
func get_all_liked_statuses(db_conn *sql.DB) []models.Status {
	rows, err := db_conn.Query(`SELECT statuses.id, statuses.text 
		FROM favourites 
		LEFT JOIN statuses ON favourites.status_id=statuses.id 
		LEFT JOIN accounts ON favourites.account_id=accounts.id
		WHERE accounts.domain IS NULL;`)
	if err != nil {
		panic(err)
	}
	var statuses []models.Status
	for rows.Next() {
		status := models.Status{}
		rows.Scan(&status.ID, &status.Content)
		statuses = append(statuses, status)
	}
	return statuses
}

// gets all statuses that were rebloged by a local user
func get_all_rebloged_statuses(db_conn *sql.DB) []models.Status {
	rows, err := db_conn.Query(`SELECT original.id, original.text 
		FROM statuses 
		LEFT JOIN statuses original ON statuses.reblog_of_id = original.id
		LEFT JOIN accounts ON statuses.account_id=accounts.id
		WHERE accounts.domain IS NULL AND statuses.reblog_of_id IS NOT NULL;`)
	if err != nil {
		panic(err)
	}
	var statuses []models.Status
	for rows.Next() {
		status := models.Status{}
		rows.Scan(&status.ID, &status.Content)
		statuses = append(statuses, status)
	}
	return statuses
}

// gets embedding for previous likes and reblogs
func main() {
	db_conn, err := get_postgres_conn()
	if err != nil {
		log.Println("Error connecting to db ")
		return
	}
	liked_statuses := get_all_liked_statuses(db_conn)
	referenced_liked_statuses := make([]*models.Status, len(liked_statuses))
	for i, status := range liked_statuses {
		referenced_liked_statuses[i] = &status
	}
	utils.GetStatusEmbeddingBulk(referenced_liked_statuses, db_conn)


	rebloged_statuses := get_all_rebloged_statuses(db_conn)
	referenced_rebloged_statuses := make([]*models.Status, len(rebloged_statuses))
	for i, status := range rebloged_statuses {
		referenced_rebloged_statuses[i] = &status
	}
	utils.GetStatusEmbeddingBulk(referenced_rebloged_statuses, db_conn)
}