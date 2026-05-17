package handler

import (
	"errors"
	"fmt"
)

// StartContainerRequest represents a request to start a sandbox container.
type StartContainerRequest struct {
	ContainerID string `json:"container_id"` // ID of the container to start
	Code        string `json:"code"`         // Python code to execute
	Timeout     int    `json:"timeout"`      // Execution timeout in seconds (0-60)
}

// StartContainerResponse represents the response from starting a container.
type StartContainerResponse struct {
	Status string `json:"status"` // Status of the container (e.g., "running")
}

// Validate validates the StartContainerRequest fields.
// Returns an error if ContainerID or Code is empty, or if Timeout is outside [0, 60].
func (r *StartContainerRequest) Validate() error {
	if r.ContainerID == "" {
		return errors.New("container ID cannot be empty")
	}
	if r.Code == "" {
		return errors.New("code cannot be empty")
	}
	if r.Timeout < 0 || r.Timeout > 60 {
		return fmt.Errorf("timeout must be between 0 and 60 seconds, got %d", r.Timeout)
	}
	return nil
}
