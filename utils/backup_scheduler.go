package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type BackupType string

const (
	FullBackupType  BackupType = "full"
	DeltaBackupType BackupType = "delta"
	SQLBackupType   BackupType = "sql"
)

// BackupConfig holds scheduling and backup options
type BackupConfig struct {
	DBPath           string
	BackupRoot       string
	Interval         time.Duration
	MaintenanceStart time.Time
	MaintenanceEnd   time.Time
	BackupTypes      []BackupType
	PartialTables    []string // for partial/module backups
}

// ScheduleBackups runs backups at the configured interval and window
func ScheduleBackups(cfg BackupConfig, stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				if now.After(cfg.MaintenanceStart) && now.Before(cfg.MaintenanceEnd) {
					for _, btype := range cfg.BackupTypes {
						backupDir := filepath.Join(cfg.BackupRoot, now.Format("2006/01/02"), string(btype))
						os.MkdirAll(backupDir, 0755)
						backupPath := filepath.Join(backupDir, fmt.Sprintf("backup_%s.db", now.Format("150405")))
						switch btype {
						case FullBackupType:
							FullBackup(cfg.DBPath, backupPath)
						case SQLBackupType:
							SQLDump(cfg.DBPath, backupPath+".sql", cfg.PartialTables)
						case DeltaBackupType:
							DeltaBackup(cfg.DBPath, cfg.DBPath+"-wal", backupPath+".wal")
						}
					}
				}
			case <-stopCh:
				return
			}
		}
	}()
}

// SQLDump creates a SQL dump of the whole DB or specific tables
func SQLDump(dbPath, outPath string, tables []string) error {
	args := []string{dbPath, ".dump"}
	if len(tables) > 0 {
		args = append(args, tables...)
	}
	cmd := exec.Command("sqlite3", args...)
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	cmd.Stdout = out
	return cmd.Run()
}
