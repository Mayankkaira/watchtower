package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now()

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "healthy",
		"uptime":     time.Since(startTime).String(),
		"go_version": runtime.Version(),
		"goroutines": runtime.NumGoroutine(),
		"memory_mb":  mem.Alloc / 1024 / 1024,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
