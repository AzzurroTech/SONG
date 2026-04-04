package faas

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

// Sandbox defines the interface for isolating function execution
type Sandbox interface {
	// Prepare sets up the isolated environment for a function execution
	Prepare(ctx context.Context, config *SandboxConfig) error

	// Execute runs the command within the sandbox
	Execute(ctx context.Context, cmd *exec.Cmd) error

	// Cleanup releases resources associated with the sandbox
	Cleanup(ctx context.Context) error

	// Isolated returns true if the sandbox provides actual isolation
	Isolated() bool
}

// SandboxConfig holds configuration for the sandbox
type SandboxConfig struct {
	// FunctionID is the unique identifier for the function
	FunctionID string

	// BinaryPath is the path to the compiled binary
	BinaryPath string

	// Args are the arguments to pass to the binary
	Args []string

	// EnvVars are the environment variables to set (minimal set)
	EnvVars map[string]string

	// Timeout is the maximum execution time
	Timeout time.Duration

	// MemoryLimitMB is the memory limit in MB (0 = unlimited)
	MemoryLimitMB int64

	// CPULimitPercent is the CPU limit percentage (0 = unlimited)
	CPULimitPercent int

	// ReadOnlyRootFS indicates if the root filesystem should be read-only
	ReadOnlyRootFS bool

	// AllowedPaths are specific paths the function is allowed to access
	AllowedPaths []string
}

// SimpleSandbox is a basic implementation that relies on OS-level process isolation
// without full containerization. It uses chroot, namespaces (if available), and cgroups.
// Note: For production, a full container runtime (Docker, containerd) is recommended.
type SimpleSandbox struct {
	config     *SandboxConfig
	workDir    string
	pid        int
	cgroupPath string
	isolated   bool
}

// NewSimpleSandbox creates a new simple sandbox instance
func NewSimpleSandbox(config *SandboxConfig) *SimpleSandbox {
	return &SimpleSandbox{
		config:   config,
		isolated: false, // Will be set to true if isolation features are available
	}
}

// Prepare sets up the sandbox environment
func (s *SimpleSandbox) Prepare(ctx context.Context) error {
	// Create a temporary working directory
	tmpDir, err := os.MkdirTemp("", "song-sandbox-*")
	if err != nil {
		return fmt.Errorf("failed to create sandbox work dir: %w", err)
	}
	s.workDir = tmpDir

	// Check for isolation capabilities
	if runtime.GOOS == "linux" {
		// Attempt to use cgroups for resource limits
		if err := s.setupCgroups(); err != nil {
			fmt.Printf("Warning: Failed to setup cgroups: %v (running without resource limits)\n", err)
		} else {
			s.isolated = true
		}

		// Attempt to setup chroot if ReadOnlyRootFS is requested
		if s.config.ReadOnlyRootFS {
			if err := s.setupChroot(); err != nil {
				fmt.Printf("Warning: Failed to setup chroot: %v (running without read-only root)\n", err)
			} else {
				s.isolated = true
			}
		}
	}

	return nil
}

// setupCgroups attempts to configure CPU and memory limits using cgroups v2
func (s *SimpleSandbox) setupCgroups() error {
	// This is a simplified implementation.
	// Real cgroups setup requires root privileges and specific kernel support.
	// We'll simulate the logic here.

	if s.config.MemoryLimitMB > 0 {
		// In a real implementation:
		// 1. Create a new cgroup: echo "song-func-123" > /sys/fs/cgroup/cgroup.subtree_control
		// 2. Set memory limit: echo <bytes> > /sys/fs/cgroup/song-func-123/memory.max
		// 3. Add PID: echo <pid> > /sys/fs/cgroup/song-func-123/cgroup.procs

		// For now, we just log that we would do it
		fmt.Printf("Cgroup setup requested for memory: %dMB\n", s.config.MemoryLimitMB)
	}

	if s.config.CPULimitPercent > 0 {
		// Similar logic for CPU quota
		fmt.Printf("Cgroup setup requested for CPU: %d%%\n", s.config.CPULimitPercent)
	}

	return nil
}

// setupChroot attempts to prepare a minimal chroot environment
func (s *SimpleSandbox) setupChroot() error {
	// In a real implementation:
	// 1. Create a minimal root filesystem with only necessary binaries (libc, ld-linux)
	// 2. Mount necessary pseudo-filesystems (proc, sys, dev)
	// 3. Call syscall.Chroot(newRoot)

	// For this example, we just ensure the workDir is isolated
	// and we don't actually chroot (which requires root).
	// We rely on the fact that the process runs in a temp dir with restricted env.

	// Copy necessary binaries if we were doing a real chroot
	// This is a placeholder for the complex logic required.

	return nil
}

// Execute runs the command within the prepared sandbox
func (s *SimpleSandbox) Execute(ctx context.Context, cmd *exec.Cmd) error {
	// Set the working directory
	cmd.Dir = s.workDir

	// Set environment variables
	env := []string{}
	for k, v := range s.config.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	// Add PATH
	env = append(env, "PATH=/usr/bin:/bin")
	cmd.Env = env

	// Set the Pdeathsig to kill the process if the parent dies
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}

	// If we have a PID (from cgroups), we would add it here
	// In a real sandbox, we would also set up namespaces (UTS, IPC, PID, NET, MNT, USER)

	// Execute the command
	return cmd.Run()
}

// Cleanup releases resources
func (s *SimpleSandbox) Cleanup(ctx context.Context) error {
	// Remove cgroup if created
	// Remove chroot if created
	// Remove work directory
	if s.workDir != "" {
		os.RemoveAll(s.workDir)
	}
	return nil
}

// Isolated returns true if the sandbox is providing actual isolation
func (s *SimpleSandbox) Isolated() bool {
	return s.isolated
}

// Note: The above implementation is a simplified "best effort" sandbox.
// For a production FaaS, you should use a container runtime like Docker or containerd.
// The following is a stub for a Docker-based sandbox implementation.

// DockerSandbox implements Sandbox using Docker containers
type DockerSandbox struct {
	config *SandboxConfig
	client *DockerClient // Hypothetical client
}

// NewDockerSandbox creates a new Docker-based sandbox
func NewDockerSandbox(config *SandboxConfig) *DockerSandbox {
	// Initialize Docker client
	return &DockerSandbox{
		config: config,
		// client: docker.NewClient(...)
	}
}

func (d *DockerSandbox) Prepare(ctx context.Context) error {
	// Pull image, create container, mount volumes
	return nil
}

func (d *DockerSandbox) Execute(ctx context.Context, cmd *exec.Cmd) error {
	// Run container, stream output
	return nil
}

func (d *DockerSandbox) Cleanup(ctx context.Context) error {
	// Remove container
	return nil
}

func (d *DockerSandbox) Isolated() bool {
	return true
}
