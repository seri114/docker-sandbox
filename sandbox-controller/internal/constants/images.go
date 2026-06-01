package constants

// Docker image constants for consistent usage across the codebase
const (
	// DefaultSandboxImage is the default sandbox Docker image for production
	// Built from sandbox-controller/docker/Dockerfile (runtime target)
	// Lightweight image with Python only (~50MB)
	DefaultSandboxImage = "sandbox:runtime"
)
