package docker

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/seri114/docker-sandbox/config"
)

// getDockerHost returns the Docker socket path, defaulting to the standard path
// or using DOCKER_HOST environment variable if set
func getDockerHost() string {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return host
	}
	return "unix:///var/run/docker.sock"
}

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		DockerHost:     getDockerHost(),
		RequestTimeout: 10 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Client() == nil {
		t.Fatal("expected underlying docker client to be initialized")
	}

	// Test Close
	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestContext(t *testing.T) {
	cfg := &config.Config{
		DockerHost:     "unix:///var/run/docker.sock",
		RequestTimeout: 5 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.Context()
	defer cancel()

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if deadline, ok := ctx.Deadline(); !ok {
		t.Fatal("expected context to have deadline")
	} else if time.Until(deadline) > 6*time.Second {
		t.Fatal("expected deadline to be within timeout duration")
	}
}

func TestCreateContainer(t *testing.T) {
	// Skip if Docker is not available
	cfg := &config.Config{
		DockerHost:     getDockerHost(),
		RequestTimeout: 10 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.Context()
	defer cancel()

	config := &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"echo", "hello"},
	}
	hostConfig := &container.HostConfig{
		NetworkMode: "none",
	}

	id, err := client.CreateContainer(ctx, config, hostConfig)
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty container ID")
	}

	// Clean up
	if err := client.RemoveContainer(ctx, id, true); err != nil {
		t.Errorf("RemoveContainer failed: %v", err)
	}
}

func TestStartContainer(t *testing.T) {
	// Skip if Docker is not available
	cfg := &config.Config{
		DockerHost:     getDockerHost(),
		RequestTimeout: 10 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.Context()
	defer cancel()

	config := &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"echo", "hello"},
	}
	hostConfig := &container.HostConfig{
		NetworkMode: "none",
	}

	id, err := client.CreateContainer(ctx, config, hostConfig)
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
	}
	defer func() {
		if err := client.RemoveContainer(ctx, id, true); err != nil {
			t.Errorf("RemoveContainer failed: %v", err)
		}
	}()

	if err := client.StartContainer(ctx, id); err != nil {
		t.Skipf("Docker daemon not available: %v", err)
	}

	// Verify container is running
	inspect, err := client.Client().ContainerInspect(ctx, id)
	if err != nil {
		t.Fatalf("ContainerInspect failed: %v", err)
	}
	if !inspect.State.Running && inspect.State.Status != "exited" {
		t.Errorf("expected container to be running or exited, got status: %s", inspect.State.Status)
	}
}

func TestContainerLogs(t *testing.T) {
	// Skip if Docker is not available
	cfg := &config.Config{
		DockerHost:     getDockerHost(),
		RequestTimeout: 10 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.Context()
	defer cancel()

	config := &container.Config{
		Image:        "alpine:latest",
		Cmd:          []string{"sh", "-c", "echo stdout && echo stderr >&2"},
		AttachStdout: true,
		AttachStderr: true,
	}
	hostConfig := &container.HostConfig{
		NetworkMode: "none",
	}

	id, err := client.CreateContainer(ctx, config, hostConfig)
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
	}
	defer func() {
		if err := client.RemoveContainer(ctx, id, true); err != nil {
			t.Errorf("RemoveContainer failed: %v", err)
		}
	}()

	if err := client.StartContainer(ctx, id); err != nil {
		t.Skipf("Docker daemon not available: %v", err)
	}

	// Test logs (follow=false)
	reader, err := client.ContainerLogs(ctx, id, false)
	if err != nil {
		t.Fatalf("ContainerLogs failed: %v", err)
	}
	defer reader.Close()

	// Read some data
	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}
	if n == 0 {
		t.Fatal("expected to read some log data")
	}

	// Test logs (follow=true) - just verify it returns a reader
	followReader, err := client.ContainerLogs(ctx, id, true)
	if err != nil {
		t.Fatalf("ContainerLogs with follow=true failed: %v", err)
	}
	defer followReader.Close()

	if followReader == nil {
		t.Fatal("expected non-nil reader for follow=true")
	}
}
