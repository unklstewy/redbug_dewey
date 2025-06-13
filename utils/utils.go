// Package utils contains utility functions for the application.
package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes the SQLite3 database and returns the connection
func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	return db
}

// HealthCheck runs DB integrity and stats queries
func HealthCheck(db *sql.DB) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	var integrity string
	err := db.QueryRow("PRAGMA integrity_check;").Scan(&integrity)
	if err != nil {
		return nil, err
	}
	stats["integrity_ok"] = (integrity == "ok")

	var dbSize int64
	fileInfo, err := os.Stat("dewey.db")
	if err == nil {
		dbSize = fileInfo.Size()
	}
	stats["db_size"] = dbSize

	var lastVacuum string
	db.QueryRow("PRAGMA auto_vacuum;").Scan(&lastVacuum)
	stats["last_vacuum"] = lastVacuum

	var walStatus string
	db.QueryRow("PRAGMA journal_mode;").Scan(&walStatus)
	stats["wal_status"] = walStatus

	tableCounts := make(map[string]int)
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table';")
	if err == nil {
		for rows.Next() {
			var table string
			rows.Scan(&table)
			var count int
			q := fmt.Sprintf("SELECT COUNT(*) FROM %s;", table)
			db.QueryRow(q).Scan(&count)
			tableCounts[table] = count
		}
		rows.Close()
	}
	jsonCounts, _ := json.Marshal(tableCounts)
	stats["table_counts"] = string(jsonCounts)

	return stats, nil
}
