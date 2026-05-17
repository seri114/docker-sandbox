package handler

import (
	"testing"
)

func TestStartContainerRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &StartContainerRequest{
			ContainerID: "container-123",
			Code:        "print('hello')",
			Timeout:     30,
		}

		if err := req.Validate(); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("empty container id", func(t *testing.T) {
		req := &StartContainerRequest{
			ContainerID: "",
			Code:        "print('hello')",
			Timeout:     30,
		}

		if err := req.Validate(); err == nil {
			t.Error("expected error for empty ContainerID, got nil")
		}
	})

	t.Run("empty code", func(t *testing.T) {
		req := &StartContainerRequest{
			ContainerID: "container-123",
			Code:        "",
			Timeout:     30,
		}

		if err := req.Validate(); err == nil {
			t.Error("expected error for empty Code, got nil")
		}
	})

	t.Run("timeout below zero", func(t *testing.T) {
		req := &StartContainerRequest{
			ContainerID: "container-123",
			Code:        "print('hello')",
			Timeout:     -1,
		}

		if err := req.Validate(); err == nil {
			t.Error("expected error for negative Timeout, got nil")
		}
	})

	t.Run("timeout above 60", func(t *testing.T) {
		req := &StartContainerRequest{
			ContainerID: "container-123",
			Code:        "print('hello')",
			Timeout:     61,
		}

		if err := req.Validate(); err == nil {
			t.Error("expected error for Timeout > 60, got nil")
		}
	})

	t.Run("timeout at boundary 0", func(t *testing.T) {
		req := &StartContainerRequest{
			ContainerID: "container-123",
			Code:        "print('hello')",
			Timeout:     0,
		}

		if err := req.Validate(); err != nil {
			t.Errorf("expected no error for Timeout=0, got %v", err)
		}
	})

	t.Run("timeout at boundary 60", func(t *testing.T) {
		req := &StartContainerRequest{
			ContainerID: "container-123",
			Code:        "print('hello')",
			Timeout:     60,
		}

		if err := req.Validate(); err != nil {
			t.Errorf("expected no error for Timeout=60, got %v", err)
		}
	})
}

func TestStartContainerResponse(t *testing.T) {
	resp := &StartContainerResponse{
		Status: "running",
	}

	if resp.Status != "running" {
		t.Errorf("expected Status 'running', got '%s'", resp.Status)
	}
}
