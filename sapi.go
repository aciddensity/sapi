package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"
)

// UptimeHandler returns system uptime
func UptimeHandler(w http.ResponseWriter, r *http.Request) {
	var sysinfo syscall.Sysinfo_t
	if err := syscall.Sysinfo(&sysinfo); err != nil {
		http.Error(w, "Failed to get uptime", http.StatusInternalServerError)
		return
	}
	response := map[string]interface{}{
		"uptime_seconds": sysinfo.Uptime,
	}
	jsonResponse(w, response)
}

// DiskUsageHandler returns disk usage of root filesystem
func DiskUsageHandler(w http.ResponseWriter, r *http.Request) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		http.Error(w, "Failed to get disk usage", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"total_bytes":  stat.Blocks * uint64(stat.Bsize),
		"free_bytes":   stat.Bfree * uint64(stat.Bsize),
		"used_bytes":   (stat.Blocks - stat.Bfree) * uint64(stat.Bsize),
		"used_percent": float64(stat.Blocks-stat.Bfree) / float64(stat.Blocks) * 100,
	}
	jsonResponse(w, response)
}

// OSReleaseHandler returns contents of /etc/os-release
func OSReleaseHandler(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		http.Error(w, "Failed to read /etc/os-release", http.StatusInternalServerError)
		return
	}

	// Parse key-value pairs from os-release file
	osInfo := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			keyVal := splitKeyValue(line)
			if keyVal != nil {
				osInfo[keyVal[0]] = keyVal[1]
			}
		}
	}

	jsonResponse(w, osInfo)
}

// jsonResponse is a helper to write JSON responses
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// splitKeyValue splits "KEY=VALUE" into a slice
func splitKeyValue(s string) []string {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) == 2 {
		return parts
	}
	return nil
}

func main() {
	http.HandleFunc("/uptime", UptimeHandler)
	http.HandleFunc("/diskusage", DiskUsageHandler)
	http.HandleFunc("/os-release", OSReleaseHandler)

	fmt.Println("Server started on :8069")
	http.ListenAndServe(":8069", nil)
}
