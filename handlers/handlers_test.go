package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/unklstewy/redbug_dewey/utils"
)

func TestManufacturerCRUD(t *testing.T) {
	// Sequential for reliability
	db := utils.InitDB(":memory:")
	defer db.Close()
	utils.CreateTables(db)

	// Create
	id, err := CreateManufacturer(db, "TestCo")
	if err != nil {
		t.Fatalf("CreateManufacturer failed: %v", err)
	}

	// Read
	m, err := GetManufacturer(db, int(id))
	if err != nil || m.Name != "TestCo" {
		t.Fatalf("GetManufacturer failed: %v", err)
	}

	// Update
	err = UpdateManufacturer(db, int(id), "NewName")
	if err != nil {
		t.Fatalf("UpdateManufacturer failed: %v", err)
	}
	m, _ = GetManufacturer(db, int(id))
	if m.Name != "NewName" {
		t.Fatalf("Update did not persist")
	}

	// Delete
	err = DeleteManufacturer(db, int(id))
	if err != nil {
		t.Fatalf("DeleteManufacturer failed: %v", err)
	}
	_, err = GetManufacturer(db, int(id))
	if err == nil {
		t.Fatalf("Manufacturer not deleted")
	}
}

func TestUserAdminFunctions(t *testing.T) {
	// Sequential for reliability
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	// Create user table for test
	_, err = db.Exec(`CREATE TABLE user (id INTEGER PRIMARY KEY, username TEXT, password_hash TEXT, role_id INTEGER, locked BOOLEAN, revoked BOOLEAN, last_login TEXT);`)
	if err != nil {
		t.Fatalf("failed to create user table: %v", err)
	}
	// Create user
	uid, err := CreateUser(db, "testuser", "testpass", 1)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if uid == 0 {
		t.Fatal("expected non-zero user id")
	}
	// Lock user
	err = LockUser(db, "testuser")
	if err != nil {
		t.Errorf("failed to lock user: %v", err)
	}
	// Unlock user
	err = UnlockUser(db, "testuser")
	if err != nil {
		t.Errorf("failed to unlock user: %v", err)
	}
	// Revoke user
	err = RevokeUser(db, "testuser")
	if err != nil {
		t.Errorf("failed to revoke user: %v", err)
	}
	// Unrevoke user
	err = UnrevokeUser(db, "testuser")
	if err != nil {
		t.Errorf("failed to unrevoke user: %v", err)
	}
	// Reset password
	err = ResetUserPassword(db, "testuser", "newpass")
	if err != nil {
		t.Errorf("failed to reset password: %v", err)
	}
	// Remove user
	err = RemoveUser(db, "testuser")
	if err != nil {
		t.Errorf("failed to remove user: %v", err)
	}
}

func TestUserAdminFunctionsIterative(t *testing.T) {
	t.Parallel() // This test is performance-bound and safe to parallelize
	for i := 0; i < 100; i++ {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatalf("cycle %d: failed to open db: %v", i, err)
		}
		_, err = db.Exec(`CREATE TABLE user (id INTEGER PRIMARY KEY, username TEXT, password_hash TEXT, role_id INTEGER, locked BOOLEAN, revoked BOOLEAN, last_login TEXT);`)
		if err != nil {
			t.Fatalf("cycle %d: failed to create user table: %v", i, err)
		}
		uid, err := CreateUser(db, "testuser", "testpass", 1)
		if err != nil {
			t.Fatalf("cycle %d: failed to create user: %v", i, err)
		}
		if uid == 0 {
			t.Fatalf("cycle %d: expected non-zero user id", i)
		}
		iCopy := i   // capture loop variable
		dbCopy := db // capture db for closure
		t.Run(fmt.Sprintf("LockUnlockRevokeCycle_%d", iCopy), func(t *testing.T) {
			t.Parallel()
			defer dbCopy.Close()
			err = LockUser(dbCopy, "testuser")
			if err != nil {
				t.Errorf("cycle %d: failed to lock user: %v", iCopy, err)
			}
			err = UnlockUser(dbCopy, "testuser")
			if err != nil {
				t.Errorf("cycle %d: failed to unlock user: %v", iCopy, err)
			}
			err = RevokeUser(dbCopy, "testuser")
			if err != nil {
				t.Errorf("cycle %d: failed to revoke user: %v", iCopy, err)
			}
			err = UnrevokeUser(dbCopy, "testuser")
			if err != nil {
				t.Errorf("cycle %d: failed to unrevoke user: %v", iCopy, err)
			}
			err = ResetUserPassword(dbCopy, "testuser", "newpass")
			if err != nil {
				t.Errorf("cycle %d: failed to reset password: %v", iCopy, err)
			}
			err = RemoveUser(dbCopy, "testuser")
			if err != nil {
				t.Errorf("cycle %d: failed to remove user: %v", iCopy, err)
			}
		})
		// Do not close db here; subtest is responsible
	}
}

func TestTeamFunctions(t *testing.T) {
	// Sequential for reliability
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE user (id INTEGER PRIMARY KEY, username TEXT, password_hash TEXT, role_id INTEGER, locked BOOLEAN, revoked BOOLEAN, last_login TEXT);`)
	if err != nil {
		t.Fatalf("failed to create user table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE role (id INTEGER PRIMARY KEY, name TEXT);`)
	if err != nil {
		t.Fatalf("failed to create role table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE team (id INTEGER PRIMARY KEY, name TEXT, leader_id INTEGER REFERENCES user(id));`)
	if err != nil {
		t.Fatalf("failed to create team table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE team_member (id INTEGER PRIMARY KEY, team_id INTEGER REFERENCES team(id), user_id INTEGER REFERENCES user(id), role_id INTEGER REFERENCES role(id));`)
	if err != nil {
		t.Fatalf("failed to create team_member table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE team_permission (id INTEGER PRIMARY KEY, team_id INTEGER REFERENCES team(id), permission_id INTEGER);`)
	if err != nil {
		t.Fatalf("failed to create team_permission table: %v", err)
	}
	// Create a user and a team
	uid, _ := CreateUser(db, "leader", "pass", 1)
	tid, err := CreateTeam(db, "TestTeam", int(uid))
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}
	// Add team member
	mid, err := AddTeamMember(db, int(tid), int(uid), 1)
	if err != nil {
		t.Errorf("failed to add team member: %v", err)
	}
	if mid == 0 {
		t.Error("expected non-zero team member id")
	}
	// Set team permission
	pid, err := SetTeamPermission(db, int(tid), 1)
	if err != nil {
		t.Errorf("failed to set team permission: %v", err)
	}
	if pid == 0 {
		t.Error("expected non-zero team permission id")
	}
	// Remove team member
	err = RemoveTeamMember(db, int(tid), int(uid))
	if err != nil {
		t.Errorf("failed to remove team member: %v", err)
	}
	// Remove team permission
	err = RemoveTeamPermission(db, int(tid), 1)
	if err != nil {
		t.Errorf("failed to remove team permission: %v", err)
	}
	// Change team leader
	err = ChangeTeamLeader(db, int(tid), int(uid))
	if err != nil {
		t.Errorf("failed to change team leader: %v", err)
	}
}

func TestParallelModuleAccess(t *testing.T) {
	t.Parallel() // This test is performance-bound and safe to parallelize
	workers := runtime.NumCPU()
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			db, err := sql.Open("sqlite3", ":memory:")
			if err != nil {
				t.Errorf("worker %d: failed to open db: %v", worker, err)
				return
			}
			defer db.Close()
			_, err = db.Exec(`CREATE TABLE user (id INTEGER PRIMARY KEY, username TEXT, password_hash TEXT, role_id INTEGER, locked BOOLEAN, revoked BOOLEAN, last_login TEXT);`)
			if err != nil {
				t.Errorf("worker %d: failed to create user table: %v", worker, err)
				return
			}
			// Create timeseries table for event capture
			err = CreateTimeseriesTable(db)
			if err != nil {
				t.Fatalf("worker %d: failed to create timeseries_event table: %v", worker, err)
				return
			}
			for j := 0; j < 100; j++ {
				username := fmt.Sprintf("user%d_%d", worker, j)
				_, err := CreateUser(db, username, "pass", 1)
				if err != nil {
					t.Errorf("worker %d: failed to create user %s: %v", worker, username, err)
				}
			}
		}(i)
	}
	wg.Wait()
	delta := time.Since(start)
	t.Logf("Parallel access with %d workers completed in %s", workers, delta)
}

func TestDatabaseMaxTransactionsPerSecond(t *testing.T) {
	const (
		initialTPS   = 100
		maxDuration  = 2 * time.Second // duration for each TPS run
		backoffRatio = 0.85            // back off by 15% on failure
		maxAttempts  = 5               // how many times to repeat the up/down search
	)
	dbPath := "tps_benchmark.db"
	defer os.Remove(dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`PRAGMA journal_mode=WAL;`)
	if err != nil {
		t.Fatalf("failed to enable WAL mode: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE user (id INTEGER PRIMARY KEY, username TEXT, password_hash TEXT, role_id INTEGER, locked BOOLEAN, revoked BOOLEAN, last_login TEXT);`)
	if err != nil {
		t.Fatalf("failed to create user table: %v", err)
	}

	tps := initialTPS
	var lastGoodTPS int
	var pattern []int
	for attempt := 0; attempt < maxAttempts; attempt++ {
		fail := false
		start := time.Now()
		var wg sync.WaitGroup
		var mu sync.Mutex
		errs := 0
		for i := 0; i < tps; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				username := fmt.Sprintf("stressuser_%d_%d", attempt, i)
				_, err := CreateUser(db, username, "pass", 1)
				if err != nil {
					mu.Lock()
					errs++
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()
		elapsed := time.Since(start)
		if errs > 0 || elapsed > maxDuration {
			fail = true
		}
		if fail {
			t.Logf("FAIL at TPS=%d (errors=%d, elapsed=%s), backing off by 15%%", tps, errs, elapsed)
			if lastGoodTPS > 0 {
				pattern = append(pattern, lastGoodTPS)
			}
			tps = int(float64(tps) * backoffRatio)
			if tps < 1 {
				t.Fatalf("TPS dropped below 1, aborting")
			}
		} else {
			t.Logf("PASS at TPS=%d (elapsed=%s)", tps, elapsed)
			lastGoodTPS = tps
			tps = int(float64(tps) * 1.15) // increase by 15%
		}
	}
	t.Logf("Repeatable TPS pattern: %v", pattern)
}

func TestDatabaseMaxStableTransactionsPerSecond(t *testing.T) {
	const (
		initialTPS      = 100
		maxDuration     = 2 * time.Second // duration for each TPS run
		maxAttempts     = 30              // allow more attempts for finer search
		stabilityTrials = 5               // how many times to test the stable TPS
	)
	dbPath := "tps_stable_benchmark.db"
	defer os.Remove(dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`PRAGMA journal_mode=WAL;`)
	if err != nil {
		t.Fatalf("failed to enable WAL mode: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE user (id INTEGER PRIMARY KEY, username TEXT, password_hash TEXT, role_id INTEGER, locked BOOLEAN, revoked BOOLEAN, last_login TEXT);`)
	if err != nil {
		t.Fatalf("failed to create user table: %v", err)
	}

	tps := initialTPS
	var lastGoodTPS int
	var stableTPS int
	fineBackoffRatio := 0.95
	for attempt := 0; attempt < maxAttempts; attempt++ {
		fail := false
		start := time.Now()
		var wg sync.WaitGroup
		var mu sync.Mutex
		errs := 0
		for i := 0; i < tps; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				username := fmt.Sprintf("stressuser_%d_%d", attempt, i)
				_, err := CreateUser(db, username, "pass", 1)
				if err != nil {
					mu.Lock()
					errs++
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()
		elapsed := time.Since(start)
		if errs > 0 || elapsed > maxDuration {
			fail = true
		}
		if fail {
			t.Logf("FAIL at TPS=%d (errors=%d, elapsed=%s), backing off by 20%%", tps, errs, elapsed)
			if lastGoodTPS > 0 {
				stableTPS = lastGoodTPS
				break
			}
			tps = int(float64(tps) * fineBackoffRatio)
			if tps < 1 {
				t.Fatalf("TPS dropped below 1, aborting")
			}
		} else {
			t.Logf("PASS at TPS=%d (elapsed=%s)", tps, elapsed)
			lastGoodTPS = tps
			tps = int(float64(tps) * 1.15) // increase by 15%
		}
	}

	if stableTPS == 0 {
		t.Fatalf("Could not find a stable TPS with zero errors")
	}
	// Minimum stable EPS required for 115200bps (assuming 1 byte per event)
	const minStableEPS = 115200
	if stableTPS < minStableEPS {
		t.Fatalf("Stable TPS %d is below required minimum for 115200bps (need at least %d)", stableTPS, minStableEPS)
	}

	t.Logf("Testing stability at TPS=%d for %d trials", stableTPS, stabilityTrials)
	for trial := 1; trial <= stabilityTrials; trial++ {
		start := time.Now()
		var wg sync.WaitGroup
		var mu sync.Mutex
		errs := 0
		for i := 0; i < stableTPS; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				username := fmt.Sprintf("stableuser_%d_%d", trial, i)
				_, err := CreateUser(db, username, "pass", 1)
				if err != nil {
					mu.Lock()
					errs++
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()
		elapsed := time.Since(start)
		if errs > 0 || elapsed > maxDuration {
			t.Errorf("Stability trial %d FAILED at TPS=%d (errors=%d, elapsed=%s)", trial, stableTPS, errs, elapsed)
		} else {
			t.Logf("Stability trial %d PASSED at TPS=%d (elapsed=%s)", trial, stableTPS, elapsed)
		}
	}
}

func TestTimeseriesEventInsertAndQuery(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	if err := CreateTimeseriesTable(db); err != nil {
		t.Fatalf("failed to create timeseries_event table: %v", err)
	}

	event := TimeseriesEvent{
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		Source:    "strace",
		Type:      "read",
		Payload:   `{"fd":3,"data":"hello"}`,
	}
	id, err := InsertTimeseriesEvent(db, event)
	if err != nil {
		t.Fatalf("failed to insert timeseries event: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero event id")
	}

	// Query for the event
	start := event.Timestamp.Add(-time.Second)
	end := event.Timestamp.Add(time.Second)
	results, err := QueryTimeseriesEvents(db, "strace", "read", start, end)
	if err != nil {
		t.Fatalf("failed to query timeseries events: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 event, got %d", len(results))
	}
	if results[0].Payload != event.Payload {
		t.Errorf("payload mismatch: got %s, want %s", results[0].Payload, event.Payload)
	}
}

func TestTimeseriesEventStress(t *testing.T) {
	const (
		initialEPS      = 500 // events per second
		maxDuration     = 2 * time.Second
		stabilityTrials = 3
	)
	dbPath := "timeseries_stress_benchmark.db"
	defer os.Remove(dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`PRAGMA journal_mode=WAL;`)
	if err != nil {
		t.Fatalf("failed to enable WAL mode: %v", err)
	}
	if err := CreateTimeseriesTable(db); err != nil {
		t.Fatalf("failed to create timeseries_event table: %v", err)
	}

	eps := initialEPS
	var lastGoodEPS int
	var stableEPS int
	fineBackoffRatio := 0.95
	var epsLog []struct {
		EPS     int
		Elapsed time.Duration
		Errors  int
	}
	for {
		fail := false
		start := time.Now()
		var wg sync.WaitGroup
		var mu sync.Mutex
		errs := 0
		for i := 0; i < eps; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				event := TimeseriesEvent{
					Timestamp: time.Now().UTC(),
					Source:    "stress",
					Type:      "test",
					Payload:   fmt.Sprintf(`{"seq":%d,"data":"payload"}`, i),
				}
				_, err := InsertTimeseriesEvent(db, event)
				if err != nil {
					mu.Lock()
					errs++
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()
		elapsed := time.Since(start)
		epsLog = append(epsLog, struct {
			EPS     int
			Elapsed time.Duration
			Errors  int
		}{eps, elapsed, errs})
		if errs > 0 || elapsed > maxDuration {
			fail = true
		}
		if fail {
			t.Logf("FAIL at EPS=%d (errors=%d, elapsed=%s), backing off by 5%%", eps, errs, elapsed)
			if lastGoodEPS > 0 {
				stableEPS = lastGoodEPS
				break
			}
			eps = int(float64(eps) * fineBackoffRatio)
			if eps < 1 {
				t.Fatalf("EPS dropped below 1, aborting")
			}
		} else {
			t.Logf("PASS at EPS=%d (elapsed=%s)", eps, elapsed)
			lastGoodEPS = eps
			eps = int(float64(eps) * 1.05) // increase by 5%
		}
	}

	t.Logf("EPS/Latency log:")
	for _, entry := range epsLog {
		t.Logf("EPS=%d, elapsed=%s, errors=%d", entry.EPS, entry.Elapsed, entry.Errors)
	}

	if stableEPS == 0 {
		t.Fatalf("Could not find a stable EPS with zero errors")
	}
	// Minimum stable EPS required for 115200bps (assuming 1 byte per event)
	const minStableEPS = 115200
	if stableEPS < minStableEPS {
		t.Fatalf("Stable EPS %d is below required minimum for 115200bps (need at least %d)", stableEPS, minStableEPS)
	}

	t.Logf("Testing stability at EPS=%d for %d trials", stableEPS, stabilityTrials)
	for trial := 1; trial <= stabilityTrials; trial++ {
		start := time.Now()
		var wg sync.WaitGroup
		var mu sync.Mutex
		errs := 0
		for i := 0; i < stableEPS; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				event := TimeseriesEvent{
					Timestamp: time.Now().UTC(),
					Source:    "stress",
					Type:      "test",
					Payload:   fmt.Sprintf(`{"seq":%d,"data":"payload"}`, i),
				}
				_, err := InsertTimeseriesEvent(db, event)
				if err != nil {
					mu.Lock()
					errs++
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()
		elapsed := time.Since(start)
		if errs > 0 || elapsed > maxDuration {
			t.Errorf("Stability trial %d FAILED at EPS=%d (errors=%d, elapsed=%s)", trial, stableEPS, errs, elapsed)
		} else {
			t.Logf("Stability trial %d PASSED at EPS=%d (elapsed=%s)", trial, stableEPS, elapsed)
		}
	}
}

func TestSimulatedCaptureIntegration(t *testing.T) {
	// Start a test HTTP server with the capture endpoints
	mux := http.NewServeMux()
	RegisterCaptureEndpoints(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	logPath := "testdata/logs/dmr_cps_read_capture.log"
	startURL := ts.URL + "/capture/start?log=" + logPath
	statusURL := ts.URL + "/capture/status"
	stopURL := ts.URL + "/capture/stop"

	// Start capture
	resp, err := http.Get(startURL)
	if err != nil {
		t.Fatalf("Failed to start capture: %v", err)
	}
	resp.Body.Close()

	// Monitor status until buffer is empty and ingestion is done
	var lastStatus string
	for i := 0; i < 100; i++ { // up to 10s
		resp, err := http.Get(statusURL)
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastStatus = string(body)
		if strings.Contains(lastStatus, "BufferLen: 0") && strings.Contains(lastStatus, "Ingesting: false") {
			break
		}
		t.Logf("Status: %s", lastStatus)
		time.Sleep(100 * time.Millisecond)
	}

	// Stop capture
	resp, err = http.Get(stopURL)
	if err != nil {
		t.Fatalf("Failed to stop capture: %v", err)
	}
	resp.Body.Close()

	t.Logf("Final status: %s", lastStatus)
}

func TestSimulatedCaptureBufferStrategies(t *testing.T) {
	logFiles := []string{
		"testdata/logs/dmr_cps_read_capture.log",
		"testdata/logs/dmr_cps_write_capture.log",
	}
	strategies := []struct {
		name     string
		strategy BufferStrategy
	}{
		{"FIFO", BufferFIFO},
		{"RED", BufferRED},
	}
	for _, logFile := range logFiles {
		for _, strat := range strategies {
			t.Run(strat.name+"/"+filepath.Base(logFile), func(t *testing.T) {
				// Clean up buffer file before test
				os.Remove("capture_buffer.dat")
				mux := http.NewServeMux()
				captureManager.bufferStrategy = strat.strategy
				RegisterCaptureEndpoints(mux)
				ts := httptest.NewServer(mux)
				defer ts.Close()

				startURL := ts.URL + "/capture/start?log=" + logFile
				statusURL := ts.URL + "/capture/status"
				stopURL := ts.URL + "/capture/stop"

				resp, err := http.Get(startURL)
				if err != nil {
					t.Fatalf("Failed to start capture: %v", err)
				}
				resp.Body.Close()

				var lastStatus string
				for i := 0; i < 100; i++ {
					resp, err := http.Get(statusURL)
					if err != nil {
						t.Fatalf("Failed to get status: %v", err)
					}
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					lastStatus = string(body)
					if strings.Contains(lastStatus, "BufferLen: 0") && strings.Contains(lastStatus, "Ingesting: false") {
						break
					}
					t.Logf("Status: %s", lastStatus)
					time.Sleep(100 * time.Millisecond)
				}

				resp, err = http.Get(stopURL)
				if err != nil {
					t.Fatalf("Failed to stop capture: %v", err)
				}
				resp.Body.Close()

				t.Logf("Final status for %s/%s: %s", strat.name, filepath.Base(logFile), lastStatus)
			})
		}
	}
}

// Handler for recording a timeseries event (could be used in an HTTP API or CLI)
// (Removed: use the implementation in handlers.go)
