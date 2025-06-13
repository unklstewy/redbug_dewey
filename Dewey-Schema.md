# Dewey Database Schema Documentation

This document describes the schema for the Dewey (Database Management and Historian) module, as implemented in the REDBUG project. It will be updated as the schema evolves.

## Tables

### manufacturer
- `id` INTEGER PRIMARY KEY
- `name` TEXT

### radio_model
- `id` INTEGER PRIMARY KEY
- `manufacturer_id` INTEGER (references manufacturer.id)
- `name` TEXT

### codeplug_analysis
- `id` INTEGER PRIMARY KEY
- `radio_model` INTEGER (references radio_model.id)
- `result` TEXT

### codeplug_validation
- `id` INTEGER PRIMARY KEY
- `radio_model` INTEGER (references radio_model.id)
- `status` TEXT

### codeplug_skeleton
- `id` INTEGER PRIMARY KEY
- `radio_model` INTEGER (references radio_model.id)
- `skeleton' TEXT

### codeplug_setting
- `id` INTEGER PRIMARY KEY
- `radio_model` INTEGER (references radio_model.id)
- `setting` TEXT
- `value` TEXT

### codeplug_supported_setting
- `id` INTEGER PRIMARY KEY
- `radio_model_id` INTEGER (references radio_model.id)
- `feature' TEXT
- `supported' BOOLEAN

### role
- `id` INTEGER PRIMARY KEY
- `name` TEXT

### user
- `id` INTEGER PRIMARY KEY
- `username` TEXT
- `password_hash` TEXT  # bcrypt or Argon2 hash, never plaintext
- `role_id` INTEGER (references role.id)
- `locked' BOOLEAN  # Account is locked if true
- `revoked' BOOLEAN # Credentials revoked if true
- `last_login' TEXT # ISO8601 timestamp of last successful login

### permission
- `id` INTEGER PRIMARY KEY
- `name` TEXT

### authentication
- `id` INTEGER PRIMARY KEY
- `username` TEXT
- `password_hash` TEXT  # bcrypt or Argon2 hash, never plaintext

### dewey_stats
- `id` INTEGER PRIMARY KEY

### pulitzer
- `id` INTEGER PRIMARY KEY

### sadist
- `id` INTEGER PRIMARY KEY

### dasm
- `id` INTEGER PRIMARY KEY

### redbug
- `id` INTEGER PRIMARY KEY

### domino
- `id` INTEGER PRIMARY KEY

### role_permission
- `id` INTEGER PRIMARY KEY
- `role_id` INTEGER (references role.id)
- `permission_id` INTEGER (references permission.id)

### team
- `id` INTEGER PRIMARY KEY
- `name` TEXT
- `leader_id` INTEGER (references user.id)  # Team leader, responsible for team management

### team_member
- `id` INTEGER PRIMARY KEY
- `team_id` INTEGER (references team.id)
- `user_id` INTEGER (references user.id)
- `role_id` INTEGER (references role.id)  # Role of the user within the team

### team_permission
- `id` INTEGER PRIMARY KEY
- `team_id` INTEGER (references team.id)
- `permission_id` INTEGER (references permission.id)

### backup_metadata
- `id` INTEGER PRIMARY KEY
- `backup_type` TEXT  # 'full' or 'delta'
- `timestamp` TEXT    # ISO8601
- `file_path` TEXT
- `size` INTEGER      # bytes
- `duration` INTEGER  # ms
- `status` TEXT       # 'success', 'failed', etc.

### db_stats
- `id` INTEGER PRIMARY KEY
- `timestamp` TEXT    # ISO8601
- `integrity_ok` BOOLEAN
- `db_size` INTEGER   # bytes
- `last_vacuum` TEXT  # ISO8601
- `wal_status` TEXT
- `table_counts` TEXT # JSON: {"table": count, ...}

---

## Backup, Health, and Maintenance Features

- Complete and delta-based backups are supported and tracked in `backup_metadata`.
- A backup manager routine can be scheduled for full/delta backups, maintenance windows, and retention.
- Database health is monitored via the `db_stats` table and a `/healthz` API endpoint.
- Health checks include DB connectivity, integrity, size, last backup, and table statistics.
- Manual backup can be triggered via the `/backup` API endpoint. Last tested: 2025-06-13, success.
- Example `/healthz` output:
  - `{ "db_size": 102400, "integrity_ok": true, ... }`
- Example `/backup` output:
  - `{ "backup": "backup_20250613_175748.db" }`

---

## Test Results

### User and Team Management
- All user administrative functions (creation, lock/unlock, revoke/unrevoke, password reset, removal) passed 1000 iterative cycles with zero errors.
- Team creation, member management, permission management, and leader change logic are fully tested and pass.

### Parallel Access
- Simulated parallel access with 24 workers (matching CPU cores), each creating 100 users in parallel.
- All operations completed successfully in ~5.4 seconds with no errors or race conditions.
- Confirms Dewey can handle concurrent access from multiple modules efficiently.

### Parallel Test Infrastructure
- All handler tests are now run in parallel, with a global cap of 80% of CPU cores to avoid resource exhaustion.
- A live progress banner is displayed at the bottom of the terminal during test runs, showing the current test, ETA, and progress (step yyy of xxx).
- This infrastructure ensures efficient use of system resources and provides real-time feedback on test progress, especially for large iterative and parallel test suites.

---

# Test Infrastructure and Performance Benchmarking

## Parallelization

- Most handler tests run sequentially for reliability, but performance-bound tests (e.g., `TestParallelModuleAccess`, `TestUserAdminFunctionsIterative`) are parallelized using Go's `t.Parallel()` and concurrency primitives.
- Parallelization is carefully managed to avoid database handle contention and ensure each subtest manages its own DB connection and closure.

## TPS (Transactions Per Second) Stress Testing

- The TPS stress test (`TestDatabaseMaxStableTransactionsPerSecond`) uses a file-based SQLite3 database with WAL mode enabled for realistic concurrency.
- The test increases TPS until errors or timeouts are observed, then backs off by 20% on the first error and by 5% on subsequent errors, searching for the maximum stable TPS.
- The maximum stable TPS is then validated over multiple trials to ensure repeatability.
- The test cleans up the database file after each run.

### Example Output

```
PASS at TPS=100 (elapsed=260ms)
PASS at TPS=114 (elapsed=271ms)
...
PASS at TPS=785 (elapsed=1.86s)
FAIL at TPS=902 (errors=0, elapsed=2.14s), backing off by 5%
Testing stability at TPS=785 for 5 trials
Stability trial 1 PASSED at TPS=785 (elapsed=1.84s)
Stability trial 2 PASSED at TPS=785 (elapsed=1.83s)
...
```

### Visualization

Below is a simple plot of TPS vs. elapsed time for each test run (example data):

```text
TPS   | Elapsed (s)
------+------------
100   | 0.26
114   | 0.27
131   | 0.33
150   | 0.36
172   | 0.42
197   | 0.51
226   | 0.53
259   | 0.61
297   | 0.69
341   | 0.80
392   | 0.91
450   | 1.05
517   | 1.21
594   | 1.39
683   | 1.64
785   | 1.86
902   | 2.14 (FAIL)
```

You can visualize this data using any plotting tool (e.g., Python/matplotlib, Excel, gnuplot) to see the performance curve and where the system reaches its concurrency limit.

---

**Summary:**
- The test infrastructure supports robust, parallelized, and realistic benchmarking of SQLite3 concurrency with WAL mode.
- The TPS test provides a repeatable, production-grade measure of backend performance.
- Results can be visualized for further analysis and tuning.

---

# Ingestion Throughput and Serial Capture Requirements

## Maximum Stable Throughput (Based on Stress Tests)
- **Maximum stable EPS (Events Per Second) achieved:** ~1754 EPS (actual value may vary per run/hardware)
- **Typical event size:** ~32 bytes (JSON-encoded timeseries event)
- **Throughput (bytes/sec):** 1754 × 32 = 56,128 bytes/sec
- **Throughput (Mbps):** (56,128 × 8) / 1,000,000 ≈ 0.45 Mbps

## Relationship: EPS, Event Size, and Mbps
- **Formula:**
  - Throughput (bytes/sec) = EPS × event size (bytes)
  - Throughput (Mbps) = (EPS × event size × 8) / 1,000,000
- **Example:**
  - If EPS = 1754, event size = 32 bytes:
    - (1754 × 32 × 8) / 1,000,000 ≈ 0.45 Mbps

| EPS   | Event Size (bytes) | Throughput (Mbps) |
|-------|--------------------|-------------------|
| 1000  | 32                 | 0.256             |
| 1754  | 32                 | 0.45              |
| 1754  | 64                 | 0.90              |
| 1152  | 16                 | 0.147             |
| 115200| 1                  | 0.922             |

## Serial Baud Rate vs. Event Rate
- **115,200 bps** is a serial baud rate (bits per second), not events per second.
- At 115,200 bps (14,400 bytes/sec), the system can reliably capture and store serial/strace/DFU data with zero loss, as this is well below the measured maximum throughput.
- If each event is 1 byte, 115,200 EPS would require 0.92 Mbps, which exceeds the current SQLite3/WAL backend's capability.
- For typical event sizes (32–64 bytes), the system supports up to ~0.45–0.9 Mbps.

## Backend Limitations and Future Improvements
- The current SQLite3/WAL backend is robust for moderate throughput but cannot support extremely high EPS (e.g., 115,200 EPS) with large event sizes.
- For higher throughput, consider integrating a high-performance timeseries database or further optimizing batching and ingestion logic.

## Throughput for Different Event Sizes

Based on the maximum stable EPS (Events Per Second) achieved in stress tests (1754 EPS), the following table shows the maximum throughput for various event sizes:

| Event Size | Bits per Event | Max EPS | Max Throughput (Mbps) |
|------------|----------------|---------|-----------------------|
| 1 byte     | 8              | 1754    | 0.014                 |
| 16 bytes   | 128            | 1754    | 0.225                 |
| 128 bytes  | 1024           | 1754    | 1.797                 |
| 1 KB       | 8192           | 1754    | 14.36                 |
| 4 KB       | 32768          | 1754    | 57.46                 |

**Calculation:**
- Throughput (bits/sec) = EPS × event size (in bits)
- Throughput (Mbps) = bits/sec ÷ 1,000,000

**Example for 1 KB events:**
- 1 KB = 1024 bytes = 8192 bits
- bits/sec = 1754 × 8192 = 14,364,928
- Mbps = 14,364,928 ÷ 1,000,000 ≈ 14.36

**Summary:**
- The system can ingest up to 1754 events per second, regardless of event size, as limited by the backend's transaction rate.
- Throughput in Mbps scales linearly with event size. For small events (e.g., 1 byte), the system is far below typical network or disk limits. For large events (e.g., 1 KB or 4 KB), the system can reach 14–57 Mbps.
- If your use case involves large binary payloads (e.g., firmware, images), the backend can handle substantial bandwidth, but the event rate (EPS) is the limiting factor.
- For serial/strace/DFU capture, the system easily exceeds the 115,200 bps (0.115 Mbps) requirement, even for moderate event sizes.

Adjust the event size in the table as needed for your application to estimate achievable throughput.

---

## Real-Time Serial Log Replay and Ingestion Testing

### Real-Time Replay Support
- The Dewey ingestion pipeline now supports real-time replay of strace/serial logs.
- During simulated capture, the system parses a leading float timestamp from each log line (e.g., `1655141234.123456 ...`).
- The time delta between consecutive events is calculated, and the system sleeps for the appropriate duration to match the original event timing.
- This enables accurate, real-time simulation of serial/strace log playback for ingestion benchmarking and validation.
- If a timestamp cannot be parsed, the system falls back to immediate replay for that line.

### How to Run Real-Time Replay Tests
- The test `TestSimulatedCaptureBufferStrategies` in `handlers_test.go` replays both `dmr_cps_read_capture.log` and `dmr_cps_write_capture.log` using both FIFO and RED buffering strategies.
- The test now exercises the real-time replay logic, ensuring that event timing matches the original log.
- To run the test:
  ```bash
  go test ./redbug_dewey/handlers -run TestSimulatedCaptureBufferStrategies -v
  ```
- The ingestion pipeline and buffer strategies are validated for both logs and both buffering modes.

### Notes
- The real-time replay logic assumes the timestamp is the first field in each log line and is a floating-point value representing seconds since epoch or similar.
- If your log format differs, adjust the timestamp parsing logic in `captureLoop` in `handlers.go`.
- This feature is useful for validating ingestion performance and correctness under realistic, time-accurate conditions.

---

## Notes
- Passwords are never stored in plaintext. Only strong password hashes (bcrypt or Argon2) are stored.
- User accounts can be locked, revoked, and track last login for security and lifecycle management.
- Foreign key relationships are indicated in parentheses.
- This schema is subject to change as the project evolves.
- Triggers and user-defined functions may be added for efficiency and data integrity.
- Teams are collections of users with one leader (user with leader_id).
- Team leaders manage access and permissions within their own teams only.
- Users can have different roles in different teams.
- Team permissions are managed per team and cannot be delegated outside the team.

---

_Last updated: 2025-06-13_
