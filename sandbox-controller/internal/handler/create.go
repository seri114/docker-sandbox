package handler

import (
	"github.com/docker/docker/api/types/container"
)

// CreateContainerRequest represents a request to create a new sandbox container.
type CreateContainerRequest struct {
	Image  string  // Container image to use (e.g., "python:3.11-alpine")
	Code   string  // Python code to execute
	Memory int64   // Memory limit in bytes
	CPU    float64 // CPU quota (0.5 = 50% of a CPU)
}

// CreateContainerResponse represents the response from creating a container.
type CreateContainerResponse struct {
	ContainerID string `json:"container_id"` // ID of the created container
}

// ToContainerConfig converts the request to a Docker container configuration.
func (r *CreateContainerRequest) ToContainerConfig() *container.Config {
	return &container.Config{
		Image: r.Image,
		Cmd:   []string{"python", "-u", "-c", r.Code},
		Tty:   false,
		User:  "nobody",
	}
}

// ToHostConfig converts the request to a Docker host configuration.
func (r *CreateContainerRequest) ToHostConfig() *container.HostConfig {
	return &container.HostConfig{
		NetworkMode: "none",
		Resources: container.Resources{
			Memory:   r.Memory,
			NanoCPUs: int64(r.CPU * 1000000000),
		},
		CapDrop: []string{"ALL"},
		Tmpfs: map[string]string{
			"/tmp": "mode=1777",
		},
	}
}
