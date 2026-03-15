package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"
)

func handleHealth(dbConn *sql.DB, dbPath string) http.HandlerFunc {
	startTime := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		dbOK := true
		if err := dbConn.Ping(); err != nil {
			dbOK = false
		}

		resp := map[string]any{
			"status":     "ok",
			"uptime":     time.Since(startTime).String(),
			"go_version": runtime.Version(),
			"goroutines": runtime.NumGoroutine(),
			"memory_mb":  float64(mem.Alloc) / 1024 / 1024,
			"db_ok":      dbOK,
		}

		// SQLite-specific stats
		if dbPath != "" {
			// Database file size
			if info, err := os.Stat(dbPath); err == nil {
				resp["db_size_mb"] = float64(info.Size()) / 1024 / 1024
			}

			// Table count
			var tableCount int
			if err := dbConn.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table'").Scan(&tableCount); err == nil {
				resp["db_tables"] = tableCount
			}

			// WAL file size
			walPath := dbPath + "-wal"
			if info, err := os.Stat(walPath); err == nil {
				resp["db_wal_size_mb"] = float64(info.Size()) / 1024 / 1024
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
