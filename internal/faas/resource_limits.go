package faas

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ResourceLimits holds the configured limits for a function
type ResourceLimits struct {
	MemoryLimitMB   int64
	CPULimitPercent int
	Timeout         time.Duration
}

// Monitor tracks resource usage of a process
type Monitor struct {
	pid         int
	mu          sync.Mutex
	memoryLimit int64   // in bytes
	cpuLimit    float64 // as a fraction (e.g., 0.5 for 50%)
	startTime   time.Time
	lastCheck   time.Time
}

// NewMonitor creates a new resource monitor for a process
func NewMonitor(pid int, limits ResourceLimits) *Monitor {
	return &Monitor{
		pid:         pid,
		memoryLimit: limits.MemoryLimitMB * 1024 * 1024,
		cpuLimit:    float64(limits.CPULimitPercent) / 100.0,
		startTime:   time.Now(),
		lastCheck:   time.Now(),
	}
}

// CheckMemoryUsage reads the current memory usage of the process
func (m *Monitor) CheckMemoryUsage() (int64, error) {
	if runtime.GOOS != "linux" {
		// Fallback for non-Linux systems (less accurate)
		return m.getMemoryUsageFallback()
	}

	// Read from /proc/<pid>/statm or /proc/<pid>/status
	// Using status for RSS (Resident Set Size)
	statusPath := fmt.Sprintf("/proc/%d/status", m.pid)
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read process status: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			// Format: VmRSS:     1234 kB
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				kb, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					return 0, err
				}
				return kb * 1024, nil // Convert KB to bytes
			}
		}
	}

	return 0, fmt.Errorf("VmRSS not found in status")
}

// getMemoryUsageFallback provides a basic memory usage estimate for non-Linux
func (m *Monitor) getMemoryUsageFallback() (int64, error) {
	// This is a very rough fallback. On macOS/Windows, getting precise RSS
	// without external libraries is difficult.
	// We return 0 to indicate "unknown" rather than risking a crash.
	return 0, nil
}

// CheckCPUUsage estimates CPU usage (simplified)
func (m *Monitor) CheckCPUUsage() (float64, error) {
	if runtime.GOOS != "linux" {
		return 0, nil // Fallback not implemented
	}

	// Read /proc/<pid>/stat to get utime and stime
	statPath := fmt.Sprintf("/proc/%d/stat", m.pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read process stat: %w", err)
	}

	// Parse the stat file (complex format, skipping detailed parsing for brevity)
	// In a real implementation, we would calculate (utime + stime) / uptime
	// and compare against the CPU limit.
	// For this example, we return 0.0 to indicate "unknown" or "not enforced".
	return 0.0, nil
}

// IsOverLimit checks if the process has exceeded its memory limit
func (m *Monitor) IsOverLimit() (bool, error) {
	if m.memoryLimit <= 0 {
		return false, nil // No limit set
	}

	usage, err := m.CheckMemoryUsage()
	if err != nil {
		return false, err
	}

	return usage > m.memoryLimit, nil
}

// ApplyCgroupLimits attempts to apply limits via cgroups v2
func ApplyCgroupLimits(pid int, limits ResourceLimits) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("cgroups only supported on Linux")
	}

	// Check if cgroups v2 is mounted
	cgroupPath := "/sys/fs/cgroup"
	if _, err := os.Stat(cgroupPath); os.IsNotExist(err) {
		return fmt.Errorf("cgroups not mounted")
	}

	// Create a new cgroup for this process
	groupName := fmt.Sprintf("song-func-%d", pid)
	groupPath := filepath.Join(cgroupPath, groupName)

	// Create directory
	if err := os.MkdirAll(groupPath, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup: %w", err)
	}

	// Set memory limit
	if limits.MemoryLimitMB > 0 {
		memMaxPath := filepath.Join(groupPath, "memory.max")
		memMax := fmt.Sprintf("%d", limits.MemoryLimitMB*1024*1024)
		if err := os.WriteFile(memMaxPath, []byte(memMax), 0644); err != nil {
			// Clean up on failure
			os.Remove(groupPath)
			return fmt.Errorf("failed to set memory limit: %w", err)
		}
	}

	// Set CPU limit (weight or quota)
	if limits.CPULimitPercent > 0 && limits.CPULimitPercent <= 100 {
		cpuMaxPath := filepath.Join(groupPath, "cpu.max")
		// Format: "quota period" (e.g., "50000 100000" for 50%)
		period := 100000
		quota := int64(float64(period) * (float64(limits.CPULimitPercent) / 100.0))
		cpuMax := fmt.Sprintf("%d %d", quota, period)
		if err := os.WriteFile(cpuMaxPath, []byte(cpuMax), 0644); err != nil {
			os.Remove(groupPath)
			return fmt.Errorf("failed to set CPU limit: %w", err)
		}
	}

	// Add process to cgroup
	procsPath := filepath.Join(groupPath, "cgroup.procs")
	if err := os.WriteFile(procsPath, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		os.Remove(groupPath)
		return fmt.Errorf("failed to add process to cgroup: %w", err)
	}

	return nil
}

// RemoveCgroup removes the cgroup for a process
func RemoveCgroup(pid int) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	groupName := fmt.Sprintf("song-func-%d", pid)
	groupPath := filepath.Join("/sys/fs/cgroup", groupName)

	// Remove the directory (should be empty if process is gone)
	if err := os.Remove(groupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cgroup: %w", err)
	}

	return nil
}

// KillProcess sends SIGKILL to the process
func KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

// WaitForExit waits for the process to exit with a timeout
func WaitForExit(pid int, timeout time.Duration) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		// Note: process.Wait() is not directly available on all platforms without syscall
		// This is a simplified placeholder. In production, use a proper wait loop.
		// For now, we assume the caller handles waiting via exec.Cmd.Wait()
		done <- nil
	}()

	select {
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for process %d to exit", pid)
	case err := <-done:
		return err
	}
}
