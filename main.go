package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/unklstewy/redbug_dewey/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User is a Gorm model for demonstration (expand as needed)
type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"unique" json:"username"`
	PasswordHash string `json:"password_hash"`
	RoleID       uint   `json:"role_id"`
	Locked       bool   `json:"locked"`
	Revoked      bool   `json:"revoked"`
	LastLogin    string `json:"last_login"`
}

// UserRole type for clarity
const (
	RoleAdmin      = 1
	RoleTeamLeader = 2
)

// AuthMiddleware extracts user info from headers (for demo; replace with real auth in prod)
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetHeader("X-User")
		roleID := c.GetHeader("X-Role")
		c.Set("username", username)
		c.Set("role_id", roleID)
		c.Next()
	}
}

// RequireRole enforces allowed roles for an endpoint
func RequireRole(allowed ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleID := c.GetString("role_id")
		for _, r := range allowed {
			if roleID == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
	}
}

func main() {
	db, err := gorm.Open(sqlite.Open("dewey.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}
	// Auto-migrate the User model (add more as needed)
	db.AutoMigrate(&User{})

	r := gin.Default()

	r.Use(AuthMiddleware())

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

	r.GET("/healthz", func(c *gin.Context) {
		sqldb, _ := db.DB()
		stats, err := utils.HealthCheck(sqldb)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, stats)
	})

	// Backup endpoint with access control
	r.POST("/backup", func(c *gin.Context) {
		roleID := c.GetString("role_id")
		if roleID == "1" { // Admin: full backup
			backupPath := "backup_" + time.Now().Format("20060102_150405") + ".db"
			err := utils.FullBackup("dewey.db", backupPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"backup": backupPath})
			return
		} else if roleID == "2" { // Team leader: partial backup only
			// For demo, just return a message (implement partial backup logic as needed)
			c.JSON(http.StatusOK, gin.H{"backup": "partial backup for your team only (not implemented)"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient privileges for backup"})
	})

	// Start backup scheduler (example config)
	stopCh := make(chan struct{})
	cfg := utils.BackupConfig{
		DBPath:           "dewey.db",
		BackupRoot:       "backups",
		Interval:         24 * time.Hour,
		MaintenanceStart: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 2, 0, 0, 0, time.Local),
		MaintenanceEnd:   time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 4, 0, 0, 0, time.Local),
		BackupTypes:      []utils.BackupType{utils.FullBackupType, utils.SQLBackupType},
		PartialTables:    []string{}, // or specify tables for partial backup
	}
	utils.ScheduleBackups(cfg, stopCh)

	r.Run(":8080")
}
