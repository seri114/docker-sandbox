package handler

import (
	"errors"
)

// StopContainerRequest represents a request to stop and remove a sandbox container.
type StopContainerRequest struct {
	ContainerID string // ID of the container to stop and remove
}

// StopContainerResponse represents the response from stopping a container.
type StopContainerResponse struct {
	Status string // Status of the container (e.g., "stopped")
}

// Validate validates the StopContainerRequest fields.
// Returns an error if ContainerID is empty.
func (r *StopContainerRequest) Validate() error {
	if r.ContainerID == "" {
		return errors.New("container ID cannot be empty")
	}
	return nil
}
