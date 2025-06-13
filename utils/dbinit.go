package utils

import (
	"database/sql"
	"fmt"
	"log"
)

// CreateTables creates the initial tables for Dewey's models
func CreateTables(db *sql.DB) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS manufacturer (id INTEGER PRIMARY KEY, name TEXT);`,
		`CREATE TABLE IF NOT EXISTS radio_model (id INTEGER PRIMARY KEY, manufacturer_id INTEGER, name TEXT);`,
		`CREATE TABLE IF NOT EXISTS codeplug_analysis (id INTEGER PRIMARY KEY, radio_model INTEGER, result TEXT);`,
		`CREATE TABLE IF NOT EXISTS codeplug_validation (id INTEGER PRIMARY KEY, radio_model INTEGER, status TEXT);`,
		`CREATE TABLE IF NOT EXISTS codeplug_skeleton (id INTEGER PRIMARY KEY, radio_model INTEGER, skeleton TEXT);`,
		`CREATE TABLE IF NOT EXISTS codeplug_setting (id INTEGER PRIMARY KEY, radio_model INTEGER, setting TEXT, value TEXT);`,
		`CREATE TABLE IF NOT EXISTS codeplug_supported_setting (id INTEGER PRIMARY KEY, radio_model_id INTEGER, feature TEXT, supported BOOLEAN);`,
		`CREATE TABLE IF NOT EXISTS role (id INTEGER PRIMARY KEY, name TEXT);`,
		`CREATE TABLE IF NOT EXISTS user (id INTEGER PRIMARY KEY, username TEXT, password TEXT, role_id INTEGER);`,
		`CREATE TABLE IF NOT EXISTS permission (id INTEGER PRIMARY KEY, name TEXT);`,
		`CREATE TABLE IF NOT EXISTS authentication (id INTEGER PRIMARY KEY, username TEXT, password TEXT);`,
		`CREATE TABLE IF NOT EXISTS dewey_stats (id INTEGER PRIMARY KEY);`,
		`CREATE TABLE IF NOT EXISTS pulitzer (id INTEGER PRIMARY KEY);`,
		`CREATE TABLE IF NOT EXISTS sadist (id INTEGER PRIMARY KEY);`,
		`CREATE TABLE IF NOT EXISTS dasm (id INTEGER PRIMARY KEY);`,
		`CREATE TABLE IF NOT EXISTS redbug (id INTEGER PRIMARY KEY);`,
		`CREATE TABLE IF NOT EXISTS domino (id INTEGER PRIMARY KEY);`,
		`CREATE TABLE IF NOT EXISTS role_permission (id INTEGER PRIMARY KEY, role_id INTEGER REFERENCES role(id), permission_id INTEGER REFERENCES permission(id));`,
		`CREATE TABLE IF NOT EXISTS team (id INTEGER PRIMARY KEY, name TEXT, leader_id INTEGER REFERENCES user(id));`,
		`CREATE TABLE IF NOT EXISTS team_member (id INTEGER PRIMARY KEY, team_id INTEGER REFERENCES team(id), user_id INTEGER REFERENCES user(id), role_id INTEGER REFERENCES role(id));`,
		`CREATE TABLE IF NOT EXISTS team_permission (id INTEGER PRIMARY KEY, team_id INTEGER REFERENCES team(id), permission_id INTEGER REFERENCES permission(id));`,
		`CREATE TABLE IF NOT EXISTS backup_metadata (id INTEGER PRIMARY KEY, backup_type TEXT, timestamp TEXT, file_path TEXT, size INTEGER, duration INTEGER, status TEXT);`,
		`CREATE TABLE IF NOT EXISTS db_stats (id INTEGER PRIMARY KEY, timestamp TEXT, integrity_ok BOOLEAN, db_size INTEGER, last_vacuum TEXT, wal_status TEXT, table_counts TEXT);`,
	}
	for _, q := range queries {
		_, err := db.Exec(q)
		if err != nil {
			log.Fatalf("Failed to create table: %v", err)
		}
	}
	fmt.Println("All tables created or already exist.")
}
