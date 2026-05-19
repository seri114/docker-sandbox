package handler

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
)

const maxCodeSize = 1 * 1024 * 1024 // 1MB

// CreateContainerRequest represents a request to create a new sandbox container.
type CreateContainerRequest struct {
	Image  string  `json:"image"`  // Container image to use (e.g., "python:3.11-alpine")
	Code   string  `json:"code"`   // Python code to execute
	Memory int64   `json:"memory"` // Memory limit in bytes
	CPU    float64 `json:"cpu"`    // CPU quota (0.5 = 50% of a CPU)
}

// CreateContainerResponse represents the response from creating a container.
type CreateContainerResponse struct {
	ContainerID string `json:"container_id"` // ID of the created container
}

// Validate validates the container creation request.
func (r *CreateContainerRequest) Validate() error {
	if r.Image == "" {
		return fmt.Errorf("image cannot be empty")
	}
	if r.Code == "" {
		return fmt.Errorf("code cannot be empty")
	}
	if len(r.Code) > maxCodeSize {
		return fmt.Errorf("code size exceeds %d bytes", maxCodeSize)
	}
	if r.Memory <= 0 {
		return fmt.Errorf("memory must be positive")
	}
	if r.CPU <= 0 || r.CPU > 1 {
		return fmt.Errorf("CPU must be between 0 and 1")
	}
	return nil
}

// ToContainerConfig converts the request to a Docker container configuration.
func (r *CreateContainerRequest) ToContainerConfig() *container.Config {
	return &container.Config{
		Image: r.Image,
		Cmd:   []string{"python", "-u", "-c", r.Code},
		Tty:   false,
		User:  "nobody",
		Env:   []string{"PYTHONUNBUFFERED=1"},
	}
}

// ToHostConfig converts the request to a Docker host configuration.
func (r *CreateContainerRequest) ToHostConfig() *container.HostConfig {
	pidsLimit := int64(100)
	memorySwap := int64(-1) // Disable memory swap

	return &container.HostConfig{
		NetworkMode: "none",
		Resources: container.Resources{
			Memory:     r.Memory,
			MemorySwap: memorySwap,
			NanoCPUs:   int64(r.CPU * 1000000000),
			PidsLimit:  &pidsLimit,
		},
		CapDrop: []string{"ALL"},
		Tmpfs: map[string]string{
			"/tmp": "mode=1777",
		},
		ReadonlyRootfs: true,
		SecurityOpt:    []string{"no-new-privileges"},
	}
}
