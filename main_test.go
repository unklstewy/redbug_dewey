package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Use the User struct from main.go, do not redeclare it here

func setupTestRouter() (*gin.Engine, *gorm.DB) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&User{})
	r := gin.Default()

	r.GET("/users", func(c *gin.Context) {
		var users []User
		db.Find(&users)
		c.JSON(http.StatusOK, users)
	})

	r.POST("/users", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		db.Create(&user)
		c.JSON(http.StatusCreated, user)
	})

	return r, db
}

func TestMain(t *testing.T) {
	// Example test
}

func TestGinGormIntegration(t *testing.T) {
	r, _ := setupTestRouter()

	// Test POST /users
	user := User{Username: "testuser", PasswordHash: "hash", RoleID: 1}
	jsonValue, _ := json.Marshal(user)
	req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	// Test GET /users
	req = httptest.NewRequest("GET", "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	body, _ := ioutil.ReadAll(w.Body)
	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		t.Fatalf("failed to unmarshal users: %v", err)
	}
	if len(users) != 1 || users[0].Username != "testuser" {
		t.Fatalf("expected user 'testuser', got %+v", users)
	}
}
