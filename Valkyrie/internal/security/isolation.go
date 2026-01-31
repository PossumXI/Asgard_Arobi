package security

import (
	"fmt"
	"os/exec"
	"runtime"
)

// isolateProcess isolates a process using OS-specific mechanisms
func (sm *ShadowMonitor) isolateProcess(pid int) error {
	switch runtime.GOOS {
	case "linux":
		return sm.isolateProcessLinux(pid)
	case "windows":
		return sm.isolateProcessWindows(pid)
	case "darwin":
		return sm.isolateProcessDarwin(pid)
	default:
		return fmt.Errorf("process isolation not supported on %s", runtime.GOOS)
	}
}

// isolateProcessLinux isolates process using cgroups on Linux
func (sm *ShadowMonitor) isolateProcessLinux(pid int) error {
	// Use systemd-run or cgroup-tools to create isolated cgroup
	cmd := exec.Command("systemd-run", "--scope", "--slice=isolated",
		fmt.Sprintf("--property=CPUQuota=10%%"),
		fmt.Sprintf("--property=MemoryLimit=100M"),
		fmt.Sprintf("--property=IOWeight=10"),
		fmt.Sprintf("--property=NetworkNamespacePath=/var/run/netns/isolated_%d", pid),
		"true")

	if err := cmd.Run(); err != nil {
		// Fallback: use cgcreate
		cmd = exec.Command("cgcreate", "-g", fmt.Sprintf("cpu,memory,blkio:isolated_%d", pid))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create cgroup: %w", err)
		}

		// Move process to cgroup
		cmd = exec.Command("cgclassify", "-g", fmt.Sprintf("cpu,memory,blkio:isolated_%d", pid), fmt.Sprintf("%d", pid))
		return cmd.Run()
	}

	return nil
}

// isolateProcessWindows isolates process using job objects on Windows
func (sm *ShadowMonitor) isolateProcessWindows(pid int) error {
	// Use PowerShell to create job object and assign process
	psScript := fmt.Sprintf(`
		$job = New-Object System.Diagnostics.ProcessJob
		$proc = Get-Process -Id %d
		$job.AddProcess($proc)
		$job.CPUAffinity = 1
		$job.MaxMemory = 100MB
	`, pid)

	cmd := exec.Command("powershell", "-Command", psScript)
	return cmd.Run()
}

// isolateProcessDarwin isolates process using launchd on macOS
func (sm *ShadowMonitor) isolateProcessDarwin(pid int) error {
	// Use launchctl to create isolated environment
	// This is a simplified implementation
	cmd := exec.Command("launchctl", "limit", fmt.Sprintf("proc/%d", pid), "100M", "100M")
	return cmd.Run()
}
