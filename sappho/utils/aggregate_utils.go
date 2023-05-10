package utils

import (
	"log"
	"github.com/michealmikeyb/mastodon/sappho/models"
	"strconv"
	"database/sql"
	"github.com/lib/pq"
	"time"
	"fmt"
)

// Bulk upsert the aggregates into the aggregates table
func UpsertAggregates(db_conn *sql.DB, aggregates []models.AggregatedCandidate) error {
	status_id_array := make([]string, len(aggregates))
	account_id_array := make([]string, len(aggregates))
	// get the status id and account ids in an array to query on
	for i, ac := range aggregates {
		status_id_array[i] = ac.Candidate.StatusId
		account_id_array[i] = ac.Candidate.AccountId
	}
	// get all the statuses and account currently on the db
	rows, err := db_conn.Query(
		`SELECT account_id, status_id FROM aggregates WHERE status_id = ANY($1) AND account_id = ANY($2)`, 
		pq.Array(status_id_array),
		pq.Array(account_id_array),
	)
	if err != nil {
		return err
	}
	// put them into two lists to either update or insert
	var candidates_in_db []models.AggregatedCandidate
	var candidates_not_in_db []models.AggregatedCandidate
	for rows.Next() {
		var status_id string
		var account_id string
		rows.Scan(&account_id, &status_id)
		for _, ac := range aggregates {
			if (status_id == ac.Candidate.StatusId && account_id == ac.Candidate.AccountId) {
				candidates_in_db = append(candidates_in_db, ac)
			}
		}
	}
	for _, ac := range aggregates {
		in_db := false
		for _, ac_in_db := range candidates_in_db {
			if (ac_in_db.Candidate.StatusId == ac.Candidate.StatusId && ac_in_db.Candidate.AccountId == ac.Candidate.AccountId) {
				in_db = true
				break
			}
		}
		if !in_db {
			candidates_not_in_db = append(candidates_not_in_db, ac)
		}
	}

	var datetime = time.Now()
	dt := datetime.Format(time.RFC3339)

	// update the ones in the db
	for _, ac := range candidates_in_db {
		_, err = db_conn.Exec(
			"UPDATE aggregates SET aggregate = $1, updated_at = $2 WHERE status_id = $3 AND account_id = $4", 
			ac.Aggregates, 
			dt,
			ac.Candidate.StatusId,
			ac.Candidate.AccountId,
		)
		if err != nil {
			return err
		}

	}

	// if all are in the db no need for insert
	if len(candidates_not_in_db) == 0 {
		log.Println("No statuses to insert, returning")
		return nil
	}
	sql_str := "INSERT INTO aggregates(status_id, account_id, aggregate, created_at, updated_at) VALUES "
	vals := []interface{}{}

	// keep track of parameter number
	parameter_number := 1
	// insert each of the aggregates into the values
	for _, ac := range candidates_not_in_db {
		account_id, err := strconv.Atoi(ac.Candidate.AccountId)
		if err != nil {
			log.Println("Error converting account id to insert aggregate ", err)
			continue
		}
		status_id, err := strconv.Atoi(ac.Candidate.StatusId)
		if err != nil {
			log.Println("Error converting status id to insert aggregate ", err)
			continue
		}
		sql_str += fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d),", 
			parameter_number, 
			parameter_number +1, 
			parameter_number +2,
			parameter_number +3,
			parameter_number +4,)
		parameter_number += 5

		vals = append(
			vals, 
			status_id, 
			account_id, 
			ac.Aggregates,
			dt,
			dt,
		)
	}
	//trim the last ,
	sql_str = sql_str[0:len(sql_str)-1]

	//prepare the statement
	stmt, err := db_conn.Prepare(sql_str)
	if err != nil {
		return err
	}

	//format all vals at once
	_, err = stmt.Exec(vals...)
	if err != nil {
		return err
	}
	return nil
}