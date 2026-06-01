package constants

// Docker image constants for consistent usage across the codebase
const (
	// DefaultSandboxImage is the default sandbox Docker image
	// Built from sandbox-controller/docker/Dockerfile
	// Used for both Python code execution and Alpine command testing
	DefaultSandboxImage = "sandbox:latest"
)
