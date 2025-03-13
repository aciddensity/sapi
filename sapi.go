package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"
  "libvirt.org/go/libvirt"
)

// VirtualMachine represents a VM's details
type VirtualMachine struct {
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	CPUUsage      uint    `json:"cpu_usage"`
	MemoryTotalMB uint64  `json:"memory_total_mb"`
	MemoryUsedMB  uint64  `json:"memory_used_mb"`
}

// GetVMsHandler retrieves information about VMs
func GetVMsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := libvirt.NewConnect("qemu:///system") // Adjust for remote connection if needed
	if err != nil {
		http.Error(w, "Failed to connect to Libvirt", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	domains, err := conn.ListAllDomains(0)
	if err != nil {
		http.Error(w, "Failed to list VMs", http.StatusInternalServerError)
		return
	}

	var vms []VirtualMachine
	for _, dom := range domains {
		name, _ := dom.GetName()
		state, _, _ := dom.GetState()
		maxMem, _ := dom.GetMaxMemory()
		memStats, _ := dom.MemoryStats(10, 0)

		var usedMem uint64
		for _, stat := range memStats {
			if stat.Tag == libvirt.DOMAIN_MEMORY_STAT_ACTUAL_BALLOON {
				usedMem = stat.Val / 1024
			}
		}

		vms = append(vms, VirtualMachine{
			Name:          name,
			Status:        domainStateToString(state),
			CPUUsage:      getVCPUCount(dom),
			MemoryTotalMB: maxMem / 1024,
			MemoryUsedMB:  usedMem,
		})
		dom.Free()
	}

	jsonResponse(w, vms)
}

// Convert domain state to human-readable string
func domainStateToString(state libvirt.DomainState) string {
	switch state {
	case libvirt.DOMAIN_RUNNING:
		return "Running"
	case libvirt.DOMAIN_BLOCKED:
		return "Blocked"
	case libvirt.DOMAIN_PAUSED:
		return "Paused"
	case libvirt.DOMAIN_SHUTDOWN:
		return "Shutdown"
	case libvirt.DOMAIN_SHUTOFF:
		return "Shutoff"
	case libvirt.DOMAIN_CRASHED:
		return "Crashed"
	default:
		return "Unknown"
	}
}

// Get the number of vCPUs assigned to a domain
func getVCPUCount(dom libvirt.Domain) uint {
	info, err := dom.GetInfo()
	if err != nil {
		return 0
	}
	return info.NrVirtCpu
}

// jsonResponse helper to send JSON responses
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
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
	http.HandleFunc("/uptime", UptimeHandler)
	http.HandleFunc("/diskusage", DiskUsageHandler)
	http.HandleFunc("/os-release", OSReleaseHandler)
	http.HandleFunc("/vms", GetVMsHandler)

	fmt.Println("Server started on :8069")
	http.ListenAndServe(":8069", nil)
}
