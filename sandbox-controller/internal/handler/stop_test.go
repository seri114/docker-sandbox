package handler

import (
	"testing"
)

func TestStopContainerRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &StopContainerRequest{
			ContainerID: "container-123",
		}

		if err := req.Validate(); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("empty container id", func(t *testing.T) {
		req := &StopContainerRequest{
			ContainerID: "",
		}

		if err := req.Validate(); err == nil {
			t.Error("expected error for empty ContainerID, got nil")
		}
	})
}

func TestStopContainerResponse(t *testing.T) {
	resp := &StopContainerResponse{
		Status: "stopped",
	}

	if resp.Status != "stopped" {
		t.Errorf("expected Status 'stopped', got '%s'", resp.Status)
	}
}
