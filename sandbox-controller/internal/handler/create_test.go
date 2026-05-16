package handler

import (
	"testing"
	"time"
)

func TestCreateContainerRequest(t *testing.T) {
	req := &CreateContainerRequest{
		Image:   "python:3.11-alpine",
		Code:    "print('hello')",
		Timeout: 30 * time.Second,
		Memory:  128 * 1024 * 1024, // 128MB
		CPU:     0.5,
	}

	// Test ToContainerConfig
	config := req.ToContainerConfig()
	if config == nil {
		t.Fatal("expected non-nil config")
	}

	if config.Image != "python:3.11-alpine" {
		t.Errorf("expected image 'python:3.11-alpine', got '%s'", config.Image)
	}

	expectedCmd := []string{"python", "-u", "-c", "print('hello')"}
	if len(config.Cmd) != len(expectedCmd) {
		t.Errorf("expected cmd %v, got %v", expectedCmd, config.Cmd)
	} else {
		for i := range expectedCmd {
			if config.Cmd[i] != expectedCmd[i] {
				t.Errorf("expected cmd[%d] '%s', got '%s'", i, expectedCmd[i], config.Cmd[i])
			}
		}
	}

	if !config.AttachStdin {
		t.Error("expected AttachStdin to be true")
	}
	if !config.AttachStdout {
		t.Error("expected AttachStdout to be true")
	}
	if !config.AttachStderr {
		t.Error("expected AttachStderr to be true")
	}
	if !config.OpenStdin {
		t.Error("expected OpenStdin to be true")
	}

	if config.User != "65532:65532" {
		t.Errorf("expected user '65532:65532', got '%s'", config.User)
	}

	// Test ToHostConfig
	hostConfig := req.ToHostConfig()
	if hostConfig == nil {
		t.Fatal("expected non-nil hostConfig")
	}

	if string(hostConfig.NetworkMode) != "none" {
		t.Errorf("expected network mode 'none', got '%s'", hostConfig.NetworkMode)
	}

	if !hostConfig.ReadonlyRootfs {
		t.Error("expected ReadonlyRootfs to be true")
	}

	if hostConfig.Resources.Memory != 128*1024*1024 {
		t.Errorf("expected memory 128MB, got %d", hostConfig.Resources.Memory)
	}

	if hostConfig.Resources.NanoCPUs != 500000000 {
		t.Errorf("expected NanoCPUs 500000000, got %d", hostConfig.Resources.NanoCPUs)
	}

	if len(hostConfig.CapDrop) == 0 || hostConfig.CapDrop[0] != "ALL" {
		t.Errorf("expected CapDrop to contain 'ALL', got %v", hostConfig.CapDrop)
	}

	tmpfs, ok := hostConfig.Tmpfs["/tmp"]
	if !ok {
		t.Error("expected /tmp in Tmpfs")
	} else if tmpfs != "mode=1777" {
		t.Errorf("expected Tmpfs /tmp to be 'mode=1777', got '%s'", tmpfs)
	}
}

func TestCreateContainerResponse(t *testing.T) {
	resp := &CreateContainerResponse{
		ContainerID: "test-container-123",
	}

	if resp.ContainerID != "test-container-123" {
		t.Errorf("expected ContainerID 'test-container-123', got '%s'", resp.ContainerID)
	}
}
