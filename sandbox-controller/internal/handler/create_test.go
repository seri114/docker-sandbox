package handler

import (
	"testing"
)

func TestCreateContainerRequest(t *testing.T) {
	req := &CreateContainerRequest{
		Image:  "python:3.12-alpine",
		Code:   "print('hello')",
		Memory: 128 * 1024 * 1024, // 128MB
		CPU:    0.5,
	}

	// Test ToContainerConfig
	config := req.ToContainerConfig()
	if config == nil {
		t.Fatal("expected non-nil config")
	}

	if config.Image != "python:3.12-alpine" {
		t.Errorf("expected image 'python:3.12-alpine', got '%s'", config.Image)
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

	if config.User != "nobody" {
		t.Errorf("expected user 'nobody', got '%s'", config.User)
	}

	// Test ToHostConfig
	hostConfig := req.ToHostConfig()
	if hostConfig == nil {
		t.Fatal("expected non-nil hostConfig")
	}

	if string(hostConfig.NetworkMode) != "none" {
		t.Errorf("expected network mode 'none', got '%s'", hostConfig.NetworkMode)
	}

	if len(hostConfig.CapDrop) == 0 || hostConfig.CapDrop[0] != "ALL" {
		t.Errorf("expected CapDrop to contain 'ALL', got %v", hostConfig.CapDrop)
	}

	if hostConfig.Resources.Memory != 128*1024*1024 {
		t.Errorf("expected memory 128MB, got %d", hostConfig.Resources.Memory)
	}

	if hostConfig.Resources.NanoCPUs != 500000000 {
		t.Errorf("expected NanoCPUs 500000000, got %d", hostConfig.Resources.NanoCPUs)
	}

	tmpfs, ok := hostConfig.Tmpfs["/tmp"]
	if !ok {
		t.Error("expected /tmp in Tmpfs")
	} else if tmpfs != "mode=1777" {
		t.Errorf("expected Tmpfs /tmp to be 'mode=1777', got '%s'", tmpfs)
	}

	// Test new security settings
	if !hostConfig.ReadonlyRootfs {
		t.Error("expected ReadonlyRootfs to be true")
	}

	if len(hostConfig.SecurityOpt) == 0 || hostConfig.SecurityOpt[0] != "no-new-privileges" {
		t.Errorf("expected SecurityOpt to contain 'no-new-privileges', got %v", hostConfig.SecurityOpt)
	}

	if hostConfig.Resources.PidsLimit == nil || *hostConfig.Resources.PidsLimit != 100 {
		t.Errorf("expected PidsLimit to be 100, got %v", hostConfig.Resources.PidsLimit)
	}

	if hostConfig.Resources.MemorySwap != -1 {
		t.Errorf("expected MemorySwap to be -1 (disabled), got %d", hostConfig.Resources.MemorySwap)
	}
}

func TestCreateContainerRequestValidate(t *testing.T) {
	validRequest := &CreateContainerRequest{
		Image:  "python:3.12-alpine",
		Code:   "print('hello')",
		Memory: 128 * 1024 * 1024,
		CPU:    0.5,
	}
	if err := validRequest.Validate(); err != nil {
		t.Errorf("valid request should not return error, got: %v", err)
	}

	emptyImage := &CreateContainerRequest{
		Code:   "print('hello')",
		Memory: 128 * 1024 * 1024,
		CPU:    0.5,
	}
	if err := emptyImage.Validate(); err == nil {
		t.Error("empty image should return error")
	}

	emptyCode := &CreateContainerRequest{
		Image:  "python:3.12-alpine",
		Code:   "",
		Memory: 128 * 1024 * 1024,
		CPU:    0.5,
	}
	if err := emptyCode.Validate(); err == nil {
		t.Error("empty code should return error")
	}

	tooLargeCode := &CreateContainerRequest{
		Image:  "python:3.12-alpine",
		Code:   string(make([]byte, maxCodeSize+1)),
		Memory: 128 * 1024 * 1024,
		CPU:    0.5,
	}
	if err := tooLargeCode.Validate(); err == nil {
		t.Error("code exceeding max size should return error")
	}

	invalidMemory := &CreateContainerRequest{
		Image:  "python:3.12-alpine",
		Code:   "print('hello')",
		Memory: 0,
		CPU:    0.5,
	}
	if err := invalidMemory.Validate(); err == nil {
		t.Error("zero memory should return error")
	}

	invalidCPU := &CreateContainerRequest{
		Image:  "python:3.12-alpine",
		Code:   "print('hello')",
		Memory: 128 * 1024 * 1024,
		CPU:    1.5,
	}
	if err := invalidCPU.Validate(); err == nil {
		t.Error("CPU > 1 should return error")
	}

	zeroCPU := &CreateContainerRequest{
		Image:  "python:3.12-alpine",
		Code:   "print('hello')",
		Memory: 128 * 1024 * 1024,
		CPU:    0,
	}
	if err := zeroCPU.Validate(); err == nil {
		t.Error("CPU <= 0 should return error")
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
