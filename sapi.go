package main

import (
  "encoding/json"
  "net/http"
  "os"
  "strings"
  "syscall"
  "log/slog"
  "encoding/csv"
  "fmt"
  "flag"
)

const version = "1.0.0"

// Config struct to hold configuration options
type Config struct {
	LogFile   string
	Address   string
	Port      string
}

// LoadConfig reads the configuration from /etc/sapi.conf
func LoadConfig() (*Config, error) {
	file, err := os.Open("/etc/sapi.conf")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	reader := csv.NewReader(file)
	reader.Comma = '='

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		switch record[0] {
		case "logfile":
			config.LogFile = record[1]
		case "address":
			config.Address = record[1]
		case "port":
			config.Port = record[1]
		}
	}

	if config.Address == "" {
		config.Address = "0.0.0.0"
	}
	if config.Port == "" {
		config.Port = "8080"
	}

	return config, nil
}

// VersionHandler returns the application version
func VersionHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"version": version,
	}
	jsonResponse(w, response)
}

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
  versionFlag := flag.Bool("v", false, "Print version and exit")
  flag.BoolVar(versionFlag, "version", false, "Print version and exit")

  logFileFlag := flag.String("l", "", "Log file location")
  flag.StringVar(logFileFlag, "logfile", "", "Log file location")

  addressFlag := flag.String("a", "", "Listening address")
  flag.StringVar(addressFlag, "address", "", "Listening address")

  portFlag := flag.String("p", "", "Listening port")
  flag.StringVar(portFlag, "port", "", "Listening port")

  flag.Parse()

  if *versionFlag {
    fmt.Println("sapi version:", version)
    os.Exit(0)
  }

  config, err := LoadConfig()
  if err != nil {
    fmt.Println("Error loading config:", err)
    os.Exit(1)
  }

	if *logFileFlag != "" {
		config.LogFile = *logFileFlag
	}
	if *addressFlag != "" {
		config.Address = *addressFlag
	}
	if *portFlag != "" {
		config.Port = *portFlag
	}

  logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
  if err != nil {
    fmt.Println("Error opening log file:", err)
    os.Exit(1)
  }
  defer logFile.Close()

  logger := slog.New(slog.NewJSONHandler(logFile, nil))
  slog.SetDefault(logger)

  http.HandleFunc("/api/v1/uptime", UptimeHandler)
  http.HandleFunc("/api/v1/diskusage", DiskUsageHandler)
  http.HandleFunc("/api/v1/os-release", OSReleaseHandler)
  http.HandleFunc("/api/v1/version", VersionHandler)

  serverAddress := fmt.Sprintf("%s:%s", config.Address, config.Port)
  slog.Info("Server started", slog.String("address", serverAddress))
  http.ListenAndServe(serverAddress, nil)
}
