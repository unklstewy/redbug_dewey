package utils

import (
	"fmt"
	"io"
	"os"
	"time"
)

// FullBackup copies the SQLite DB file to a backup location
func FullBackup(dbPath, backupPath string) error {
	src, err := os.Open(dbPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// DeltaBackup is a stub for future WAL/delta backup support
func DeltaBackup(dbPath, walPath, backupPath string) error {
	// Implement WAL or .changes backup logic here
	return fmt.Errorf("Delta backup not yet implemented")
}

// ScheduleBackup runs backups at the given interval (in minutes)
func ScheduleBackup(dbPath, backupDir string, intervalMinutes int, stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				backupPath := fmt.Sprintf("%s/backup_%d.db", backupDir, time.Now().Unix())
				FullBackup(dbPath, backupPath)
			case <-stopCh:
				return
			}
		}
	}()
}
