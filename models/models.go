// Package models contains data structures and types for the application.
package models

import (
	"golang.org/x/crypto/bcrypt"
)

// Core system stubs

type Pulitzer struct{}
type SADIST struct{}
type DASM struct{}
type REDBUG struct{}
type DOMINO struct{}
type DeweyStats struct{}

type Authentication struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Manufacturer struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type RadioModel struct {
	ID             int    `json:"id"`
	ManufacturerID int    `json:"manufacturer_id"`
	Name           string `json:"name"`
}

type CodeplugAnalysis struct {
	ID         int    `json:"id"`
	RadioModel int    `json:"radio_model"`
	Result     string `json:"result"`
}

type CodeplugValidation struct {
	ID         int    `json:"id"`
	RadioModel int    `json:"radio_model"`
	Status     string `json:"status"`
}

type CodeplugSkeleton struct {
	ID         int    `json:"id"`
	RadioModel int    `json:"radio_model"`
	Skeleton   string `json:"skeleton"`
}

type CodeplugSetting struct {
	ID         int    `json:"id"`
	RadioModel int    `json:"radio_model"`
	Setting    string `json:"setting"`
	Value      string `json:"value"`
}

type CodeplugSupportedSetting struct {
	ID           int    `json:"id"`
	RadioModelID int    `json:"radio_model_id"`
	Feature      string `json:"feature"`
	Supported    bool   `json:"supported"`
}

type Role struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	RoleID       int    `json:"role_id"`
	Locked       bool   `json:"locked"`
	Revoked      bool   `json:"revoked"`
	LastLogin    string `json:"last_login"`
}

// HashPassword hashes a plaintext password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPasswordHash compares a plaintext password with a bcrypt hash
func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type Permission struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Team struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	LeaderID int    `json:"leader_id"`
}

type TeamMember struct {
	ID     int `json:"id"`
	TeamID int `json:"team_id"`
	UserID int `json:"user_id"`
	RoleID int `json:"role_id"`
}

type TeamPermission struct {
	ID           int `json:"id"`
	TeamID       int `json:"team_id"`
	PermissionID int `json:"permission_id"`
}

type BackupMetadata struct {
	ID         int    `json:"id"`
	BackupType string `json:"backup_type"`
	Timestamp  string `json:"timestamp"`
	FilePath   string `json:"file_path"`
	Size       int64  `json:"size"`
	Duration   int64  `json:"duration"`
	Status     string `json:"status"`
}

type DBStats struct {
	ID          int    `json:"id"`
	Timestamp   string `json:"timestamp"`
	IntegrityOK bool   `json:"integrity_ok"`
	DBSize      int64  `json:"db_size"`
	LastVacuum  string `json:"last_vacuum"`
	WALStatus   string `json:"wal_status"`
	TableCounts string `json:"table_counts"`
}
