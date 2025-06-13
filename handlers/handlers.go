// Package handlers contains HTTP and business logic handlers.
package handlers

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/unklstewy/redbug_dewey/models"
)

// Add a global DB handle for ingestion (for demo; in production, use a proper pool or context)
var captureDB *sql.DB

// SetCaptureDB allows setting the DB for the capture pipeline
func SetCaptureDB(db *sql.DB) {
	captureDB = db
}

// Manufacturer CRUD
func CreateManufacturer(db *sql.DB, name string) (int64, error) {
	res, err := db.Exec("INSERT INTO manufacturer (name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func GetManufacturer(db *sql.DB, id int) (*models.Manufacturer, error) {
	row := db.QueryRow("SELECT id, name FROM manufacturer WHERE id = ?", id)
	var m models.Manufacturer
	if err := row.Scan(&m.ID, &m.Name); err != nil {
		return nil, err
	}
	return &m, nil
}

func UpdateManufacturer(db *sql.DB, id int, name string) error {
	_, err := db.Exec("UPDATE manufacturer SET name = ? WHERE id = ?", name, id)
	return err
}

func DeleteManufacturer(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM manufacturer WHERE id = ?", id)
	return err
}

// User CRUD with bcrypt password hashing
func CreateUser(db *sql.DB, username, password string, roleID int) (int64, error) {
	hash, err := models.HashPassword(password)
	if err != nil {
		return 0, err
	}
	res, err := db.Exec("INSERT INTO user (username, password_hash, role_id) VALUES (?, ?, ?)", username, hash, roleID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func AuthenticateUser(db *sql.DB, username, password string) (bool, error) {
	row := db.QueryRow("SELECT password_hash FROM user WHERE username = ?", username)
	var hash string
	if err := row.Scan(&hash); err != nil {
		return false, err
	}
	return models.CheckPasswordHash(password, hash), nil
}

// Administrative functions for user management
func LockUser(db *sql.DB, username string) error {
	_, err := db.Exec("UPDATE user SET locked = 1 WHERE username = ?", username)
	return err
}

func UnlockUser(db *sql.DB, username string) error {
	_, err := db.Exec("UPDATE user SET locked = 0 WHERE username = ?", username)
	return err
}

func RevokeUser(db *sql.DB, username string) error {
	_, err := db.Exec("UPDATE user SET revoked = 1 WHERE username = ?", username)
	return err
}

func UnrevokeUser(db *sql.DB, username string) error {
	_, err := db.Exec("UPDATE user SET revoked = 0 WHERE username = ?", username)
	return err
}

func RemoveUser(db *sql.DB, username string) error {
	_, err := db.Exec("DELETE FROM user WHERE username = ?", username)
	return err
}

func ResetUserPassword(db *sql.DB, username, newPassword string) error {
	hash, err := models.HashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE user SET password_hash = ? WHERE username = ?", hash, username)
	return err
}

// Update last login timestamp
func UpdateLastLogin(db *sql.DB, username string) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec("UPDATE user SET last_login = ? WHERE username = ?", timestamp, username)
	return err
}

// Team CRUD
func CreateTeam(db *sql.DB, name string, leaderID int) (int64, error) {
	res, err := db.Exec("INSERT INTO team (name, leader_id) VALUES (?, ?)", name, leaderID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func AddTeamMember(db *sql.DB, teamID, userID, roleID int) (int64, error) {
	res, err := db.Exec("INSERT INTO team_member (team_id, user_id, role_id) VALUES (?, ?, ?)", teamID, userID, roleID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func SetTeamPermission(db *sql.DB, teamID, permissionID int) (int64, error) {
	res, err := db.Exec("INSERT INTO team_permission (team_id, permission_id) VALUES (?, ?)", teamID, permissionID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func RemoveTeamMember(db *sql.DB, teamID, userID int) error {
	_, err := db.Exec("DELETE FROM team_member WHERE team_id = ? AND user_id = ?", teamID, userID)
	return err
}

func RemoveTeamPermission(db *sql.DB, teamID, permissionID int) error {
	_, err := db.Exec("DELETE FROM team_permission WHERE team_id = ? AND permission_id = ?", teamID, permissionID)
	return err
}

func ChangeTeamLeader(db *sql.DB, teamID, newLeaderID int) error {
	_, err := db.Exec("UPDATE team SET leader_id = ? WHERE id = ?", newLeaderID, teamID)
	return err
}

// TimeseriesEvent represents a single event in the timeseries database.
type TimeseriesEvent struct {
	ID        int64     `db:"id"`
	Timestamp time.Time `db:"timestamp"`
	Source    string    `db:"source"`  // e.g., "strace", "serial", "dfu"
	Type      string    `db:"type"`    // e.g., "read", "write", "event"
	Payload   string    `db:"payload"` // JSON, text, or base64-encoded binary
}

// CreateTimeseriesTable creates the timeseries table if it does not exist.
func CreateTimeseriesTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS timeseries_event (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			source TEXT NOT NULL,
			type TEXT NOT NULL,
			payload TEXT NOT NULL
		);
	`)
	return err
}

// InsertTimeseriesEvent inserts a new event into the timeseries table.
func InsertTimeseriesEvent(db *sql.DB, event TimeseriesEvent) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO timeseries_event (timestamp, source, type, payload) VALUES (?, ?, ?, ?)`,
		event.Timestamp, event.Source, event.Type, event.Payload,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// QueryTimeseriesEvents retrieves events by source/type/time range.
func QueryTimeseriesEvents(db *sql.DB, source, eventType string, start, end time.Time) ([]TimeseriesEvent, error) {
	rows, err := db.Query(
		`SELECT id, timestamp, source, type, payload FROM timeseries_event WHERE source = ? AND type = ? AND timestamp BETWEEN ? AND ? ORDER BY timestamp`,
		source, eventType, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []TimeseriesEvent
	for rows.Next() {
		var e TimeseriesEvent
		var ts string
		if err := rows.Scan(&e.ID, &ts, &e.Source, &e.Type, &e.Payload); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		events = append(events, e)
	}
	return events, nil
}

// Handler for recording a timeseries event (for use in HTTP API, CLI, or internal calls)
func RecordTimeseriesEvent(db *sql.DB, source, eventType, payload string) (int64, error) {
	return InsertTimeseriesEvent(db, TimeseriesEvent{
		Timestamp: time.Now().UTC(),
		Source:    source,
		Type:      eventType,
		Payload:   payload,
	})
}

// CaptureBuffer defines the interface for buffering strategies
// (FIFO, RED, etc.)
type CaptureBuffer interface {
	Append([]byte) error
	ReadBatch(max int) ([][]byte, error)
	RemoveBatch(n int) error
	Len() int
	SizeBytes() int64
	Close() error
}

// FIFOBuffer implements a file-backed FIFO queue
// (current logic, refactored)
type FIFOBuffer struct {
	path string
	file *os.File
}

func NewFIFOBuffer(path string) (*FIFOBuffer, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &FIFOBuffer{path: path, file: file}, nil
}

func (b *FIFOBuffer) Append(data []byte) error {
	return writeLengthPrefixed(b.file, data)
}

func (b *FIFOBuffer) ReadBatch(max int) ([][]byte, error) {
	return readBatchFromDisk(b.path, max)
}

func (b *FIFOBuffer) RemoveBatch(n int) error {
	return removeBatchFromDisk(b.path, n)
}

func (b *FIFOBuffer) Len() int {
	batch, _ := b.ReadBatch(1000000)
	return len(batch)
}

func (b *FIFOBuffer) SizeBytes() int64 {
	fi, err := b.file.Stat()
	if err != nil {
		return 0
	}
	return fi.Size()
}

func (b *FIFOBuffer) Close() error {
	return b.file.Close()
}

// REDBuffer is a stub for Random Early Detection (future logic)
type REDBuffer struct {
	// TODO: implement RED logic (drop probability, thresholds, etc.)
	fifo *FIFOBuffer // fallback to FIFO for now
}

func NewREDBuffer(path string) (*REDBuffer, error) {
	fifo, err := NewFIFOBuffer(path)
	if err != nil {
		return nil, err
	}
	return &REDBuffer{fifo: fifo}, nil
}

func (b *REDBuffer) Append(data []byte) error {
	// TODO: implement RED drop logic
	return b.fifo.Append(data)
}
func (b *REDBuffer) ReadBatch(max int) ([][]byte, error) {
	return b.fifo.ReadBatch(max)
}
func (b *REDBuffer) RemoveBatch(n int) error {
	return b.fifo.RemoveBatch(n)
}
func (b *REDBuffer) Len() int {
	return b.fifo.Len()
}
func (b *REDBuffer) SizeBytes() int64 {
	return b.fifo.SizeBytes()
}
func (b *REDBuffer) Close() error {
	return b.fifo.Close()
}

// BufferStrategy defines available strategies
type BufferStrategy string

const (
	BufferFIFO BufferStrategy = "fifo"
	BufferRED  BufferStrategy = "red"
)

// CaptureManager manages simulated stream capture and async ingestion
var captureManager = &CaptureManager{}

type CaptureManager struct {
	mu             sync.Mutex
	buffer         [][]byte // fallback in-memory buffer (for bursts)
	file           *os.File // input log file
	bufferFilePath string   // path to buffer file
	bufferImpl     CaptureBuffer
	bufferStrategy BufferStrategy
	stopCh         chan struct{}
	stopped        bool
	ingesting      bool
	lastStatus     CaptureStatus
}

type CaptureStatus struct {
	BufferLen       int
	DiskBufferBytes int64
	Ingesting       bool
	Stopped         bool
	LastError       string
	Ingested        int
	LastUpdated     time.Time
	IngestRateEPS   float64 // events per second
	ErrorCount      int
}

// StartSimulatedCapture starts reading from a log file and buffering events
func (cm *CaptureManager) StartSimulatedCapture(logPath string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.ingesting {
		return errors.New("capture already running")
	}
	file, err := os.Open(logPath)
	if err != nil {
		return err
	}
	cm.file = file
	cm.buffer = make([][]byte, 0, 4096)
	cm.stopCh = make(chan struct{})
	cm.stopped = false
	cm.ingesting = true
	cm.lastStatus = CaptureStatus{Ingesting: true, Stopped: false, LastUpdated: time.Now()}
	// Select buffer strategy
	cm.bufferFilePath = "capture_buffer.dat"
	if cm.bufferStrategy == "red" {
		cm.bufferImpl, err = NewREDBuffer(cm.bufferFilePath)
	} else {
		cm.bufferImpl, err = NewFIFOBuffer(cm.bufferFilePath)
	}
	if err != nil {
		cm.file.Close()
		return err
	}
	go cm.captureLoop()
	go cm.ingestLoop()
	return nil
}

// StopSimulatedCapture signals the capture to stop
func (cm *CaptureManager) StopSimulatedCapture() {
	cm.mu.Lock()
	if cm.stopped {
		cm.mu.Unlock()
		return
	}
	cm.stopped = true
	if cm.stopCh != nil {
		close(cm.stopCh)
	}
	if cm.file != nil {
		cm.file.Close()
		cm.file = nil
	}
	if cm.bufferImpl != nil {
		cm.bufferImpl.Close()
		cm.bufferImpl = nil
	}
	cm.ingesting = false
	cm.mu.Unlock()
}

// GetCaptureStatus returns the current status
func (cm *CaptureManager) GetCaptureStatus() CaptureStatus {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	status := cm.lastStatus
	status.BufferLen = len(cm.buffer)
	status.Ingesting = cm.ingesting
	status.Stopped = cm.stopped
	status.LastUpdated = time.Now()
	if cm.bufferImpl != nil {
		status.DiskBufferBytes = cm.bufferImpl.SizeBytes()
	}
	return status
}

// captureLoop reads lines from the file and appends to buffer
func (cm *CaptureManager) captureLoop() {
	scanner := bufio.NewScanner(cm.file)
	var lastTimestamp float64
	var first bool = true
	for scanner.Scan() {
		select {
		case <-cm.stopCh:
			return
		default:
		}
		line := scanner.Bytes()
		// Attempt to parse a leading timestamp (float, e.g., 1655141234.123456)
		ts := 0.0
		parsed := false
		for i, b := range line {
			if b == ' ' || b == '\t' {
				tsStr := string(line[:i])
				if t, err := strconv.ParseFloat(tsStr, 64); err == nil {
					ts = t
					parsed = true
				}
				break
			}
		}
		if parsed {
			if first {
				lastTimestamp = ts
				first = false
			} else {
				delta := ts - lastTimestamp
				if delta > 0 && delta < 10 {
					time.Sleep(time.Duration(delta * float64(time.Second)))
				}
				lastTimestamp = ts
			}
		}
		cm.mu.Lock()
		if cm.bufferImpl != nil {
			cm.bufferImpl.Append(line)
		} else {
			cm.buffer = append(cm.buffer, append([]byte(nil), line...))
		}
		cm.mu.Unlock()
	}
	if err := scanner.Err(); err != nil {
		cm.mu.Lock()
		cm.lastStatus.LastError = err.Error()
		cm.mu.Unlock()
	}
}

// ingestLoop asynchronously ingests buffered events from disk into the DB
func (cm *CaptureManager) ingestLoop() {
	var lastIngested int
	var lastTime = time.Now()
	for {
		cm.mu.Lock()
		if cm.stopped {
			cm.ingesting = false
			cm.mu.Unlock()
			return
		}
		cm.mu.Unlock()
		// Read and ingest from buffer
		var batch [][]byte
		var err error
		if cm.bufferImpl != nil {
			batch, err = cm.bufferImpl.ReadBatch(256)
		} else {
			cm.mu.Lock()
			batch = cm.buffer
			cm.buffer = nil
			cm.mu.Unlock()
		}
		if err != nil {
			cm.mu.Lock()
			cm.lastStatus.LastError = err.Error()
			cm.mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if len(batch) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		// Ingest batch
		ingested := 0
		errs := 0
		if captureDB == nil {
			cm.mu.Lock()
			cm.lastStatus.LastError = "captureDB not set"
			cm.mu.Unlock()
			continue
		}
		tx, err := captureDB.Begin()
		if err != nil {
			cm.mu.Lock()
			cm.lastStatus.LastError = err.Error()
			cm.mu.Unlock()
			continue
		}
		stmt, err := tx.Prepare("INSERT INTO timeseries_event (timestamp, source, type, payload) VALUES (?, ?, ?, ?)")
		if err != nil {
			tx.Rollback()
			cm.mu.Lock()
			cm.lastStatus.LastError = err.Error()
			cm.mu.Unlock()
			continue
		}
		for _, line := range batch {
			_, err := stmt.Exec(time.Now().UTC(), "capture", "stream", string(line))
			if err != nil {
				errs++
				continue
			}
			ingested++
		}
		stmt.Close()
		err = tx.Commit()
		if err != nil {
			cm.mu.Lock()
			cm.lastStatus.LastError = err.Error()
			cm.mu.Unlock()
			continue
		}
		if cm.bufferImpl != nil {
			cm.bufferImpl.RemoveBatch(len(batch))
		}
		cm.mu.Lock()
		cm.lastStatus.Ingested += ingested
		cm.lastStatus.ErrorCount += errs
		// Calculate ingestion rate
		elapsed := time.Since(lastTime).Seconds()
		if elapsed > 0 {
			cm.lastStatus.IngestRateEPS = float64(cm.lastStatus.Ingested-lastIngested) / elapsed
			lastIngested = cm.lastStatus.Ingested
			lastTime = time.Now()
		}
		if errs > 0 {
			cm.lastStatus.LastError = fmt.Sprintf("%d ingestion errors", errs)
		}
		cm.mu.Unlock()
	}
}

// Helper: write a length-prefixed record to file
func writeLengthPrefixed(f *os.File, data []byte) error {
	var lenBuf [4]byte
	l := uint32(len(data))
	lenBuf[0] = byte(l >> 24)
	lenBuf[1] = byte(l >> 16)
	lenBuf[2] = byte(l >> 8)
	lenBuf[3] = byte(l)
	if _, err := f.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := f.Write(data)
	return err
}

// Helper: read a batch of length-prefixed records from file
func readBatchFromDisk(path string, max int) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var batch [][]byte
	for i := 0; i < max; i++ {
		var lenBuf [4]byte
		_, err := f.Read(lenBuf[:])
		if err != nil {
			break
		}
		l := (uint32(lenBuf[0]) << 24) | (uint32(lenBuf[1]) << 16) | (uint32(lenBuf[2]) << 8) | uint32(lenBuf[3])
		buf := make([]byte, l)
		_, err = f.Read(buf)
		if err != nil {
			break
		}
		batch = append(batch, buf)
	}
	return batch, nil
}

// Helper: remove N records from the start of the file
func removeBatchFromDisk(path string, n int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var offset int64
	for i := 0; i < n; i++ {
		var lenBuf [4]byte
		_, err := f.Read(lenBuf[:])
		if err != nil {
			break
		}
		l := (uint32(lenBuf[0]) << 24) | (uint32(lenBuf[1]) << 16) | (uint32(lenBuf[2]) << 8) | uint32(lenBuf[3])
		offset += 4 + int64(l)
		_, err = f.Seek(int64(l), 1)
		if err != nil {
			break
		}
	}
	// Truncate file by copying remaining data to a new file
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if offset >= fi.Size() {
		// All data consumed, just truncate
		return os.Truncate(path, 0)
	}
	// Copy remaining data
	f.Seek(offset, 0)
	rem, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer tmp.Close()
	_, err = tmp.Write(rem)
	if err != nil {
		return err
	}
	tmp.Close()
	f.Close()
	return os.Rename(tmpPath, path)
}

// HTTP Handlers
func CaptureStartHandler(w http.ResponseWriter, r *http.Request) {
	logPath := r.URL.Query().Get("log")
	if logPath == "" {
		logPath = "testdata/logs/dmr_cps_read_capture.log"
	}
	err := captureManager.StartSimulatedCapture(logPath)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("Failed to start capture: " + err.Error()))
		return
	}
	w.Write([]byte("Capture started from " + logPath + "\n"))
}

func CaptureStopHandler(w http.ResponseWriter, r *http.Request) {
	captureManager.StopSimulatedCapture()
	w.Write([]byte("Capture stopped\n"))
}

func CaptureStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := captureManager.GetCaptureStatus()
	fmt.Fprintf(w, "BufferLen: %d\nIngesting: %v\nStopped: %v\nIngested: %d\nLastError: %s\nLastUpdated: %s\nIngestRateEPS: %.2f\nErrorCount: %d\n",
		status.BufferLen, status.Ingesting, status.Stopped, status.Ingested, status.LastError, status.LastUpdated.Format(time.RFC3339), status.IngestRateEPS, status.ErrorCount)
}

// RegisterCaptureEndpoints registers the HTTP handlers for capture
func RegisterCaptureEndpoints(mux *http.ServeMux) {
	if captureDB == nil {
		db, err := sql.Open("sqlite3", ":memory:")
		if err == nil {
			CreateTimeseriesTable(db)
			SetCaptureDB(db)
		}
	}
	mux.HandleFunc("/capture/start", CaptureStartHandler)
	mux.HandleFunc("/capture/stop", CaptureStopHandler)
	mux.HandleFunc("/capture/status", CaptureStatusHandler)
}
