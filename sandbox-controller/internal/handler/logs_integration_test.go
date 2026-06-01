package handler

import (
	"bytes"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/seri114/docker-sandbox/config"
	"github.com/seri114/docker-sandbox/internal/constants"
	"github.com/seri114/docker-sandbox/internal/docker"
)

// TestLogsStreamerIntegration tests LogsStreamer with actual Docker container logs.
// This is an integration test that requires a running Docker daemon.
func TestLogsStreamerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := &config.Config{
		DockerHost:     getDockerHost(),
		RequestTimeout: 30 * time.Second,
	}

	client, err := docker.NewClient(cfg)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.Context()
	defer cancel()

	// Create a test container that outputs both stdout and stderr
	containerConfig := &container.Config{
		Image:        constants.DefaultSandboxImage,
		Cmd:          []string{"sh", "-c", "echo stdout1 && echo stderr >&2 && echo stdout2"},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}
	hostConfig := &container.HostConfig{
		NetworkMode: "none",
	}

	containerID, err := client.CreateContainer(ctx, containerConfig, hostConfig)
	if err != nil {
		t.Skipf("Failed to create container: %v", err)
	}
	defer func() {
		ctx2, cancel2 := client.Context()
		defer cancel2()
		client.RemoveContainer(ctx2, containerID, true)
	}()

	// Start the container
	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// Wait for container to finish
	for i := 0; i < 20; i++ {
		inspect, err := client.InspectContainer(ctx, containerID)
		if err != nil {
			t.Fatalf("Failed to inspect container: %v", err)
		}
		if !inspect.State.Running {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Get logs using the LogsStreamer
	logReader, err := client.ContainerLogs(ctx, containerID, false)
	if err != nil {
		t.Fatalf("Failed to get container logs: %v", err)
	}
	defer logReader.Close()

	// Read all log data into a buffer
	var logBuf bytes.Buffer
	logBuf.ReadFrom(logReader)
	logData := logBuf.Bytes()

	t.Logf("Raw log data (%d bytes): %v", len(logData), logData)

	// Now stream it through LogsStreamer
	streamer := NewLogsStreamer(bytes.NewReader(logData))
	msgCh := make(chan LogMessage, 10)

	go streamer.StreamTo(msgCh)

	// Collect messages
	var messages []LogMessage
	timeout := time.After(5 * time.Second)
collect:
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				break collect
			}
			messages = append(messages, msg)
			t.Logf("Received message: stream=%s data=%q", msg.Stream, msg.Data)
		case <-timeout:
			t.Fatal("timeout waiting for messages")
		}
	}

	// Verify we got messages
	if len(messages) == 0 {
		t.Fatalf("expected at least 1 message, got %d", len(messages))
	}

	// Verify message content
	hasStdout := false
	hasStderr := false
	for _, msg := range messages {
		if msg.Stream == "stdout" {
			hasStdout = true
		}
		if msg.Stream == "stderr" {
			hasStderr = true
		}
	}

	if !hasStdout {
		t.Error("expected at least one stdout message")
	}
	if !hasStderr {
		t.Error("expected at least one stderr message")
	}

	t.Logf("Test passed: received %d messages", len(messages))
}

// TestLogsStreamerLargeOutput tests with larger output to verify
// the size parsing works correctly with actual Docker data.
func TestLogsStreamerLargeOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := &config.Config{
		DockerHost:     getDockerHost(),
		RequestTimeout: 30 * time.Second,
	}

	client, err := docker.NewClient(cfg)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer client.Close()

	ctx, cancel := client.Context()
	defer cancel()

	// Create output larger than the buffer size (4096)
	largeOutput := string(make([]byte, 5000))
	for i := range largeOutput {
		largeOutput = largeOutput[:i] + "X"
	}

	containerConfig := &container.Config{
		Image:        constants.DefaultSandboxImage,
		Cmd:          []string{"python", "-c", "print('A' * 5000)"},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}
	hostConfig := &container.HostConfig{
		NetworkMode: "none",
	}

	containerID, err := client.CreateContainer(ctx, containerConfig, hostConfig)
	if err != nil {
		t.Skipf("Failed to create container: %v", err)
	}
	defer func() {
		ctx2, cancel2 := client.Context()
		defer cancel2()
		client.RemoveContainer(ctx2, containerID, true)
	}()

	// Start the container
	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// Wait for container to finish
	for i := 0; i < 20; i++ {
		inspect, err := client.InspectContainer(ctx, containerID)
		if err != nil {
			t.Fatalf("Failed to inspect container: %v", err)
		}
		if !inspect.State.Running {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Get logs
	logReader, err := client.ContainerLogs(ctx, containerID, false)
	if err != nil {
		t.Fatalf("Failed to get container logs: %v", err)
	}
	defer logReader.Close()

	// Stream it through LogsStreamer
	streamer := NewLogsStreamer(logReader)
	msgCh := make(chan LogMessage, 10)

	go streamer.StreamTo(msgCh)

	// Collect messages
	var messages []LogMessage
	timeout := time.After(5 * time.Second)
collect:
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				break collect
			}
			messages = append(messages, msg)
		case <-timeout:
			t.Fatal("timeout waiting for messages")
		}
	}

	if len(messages) == 0 {
		t.Fatalf("expected at least 1 message, got %d", len(messages))
	}

	// Verify the large output was received correctly
	totalLen := 0
	for _, msg := range messages {
		totalLen += len(msg.Data)
	}

	if totalLen < 5000 {
		t.Errorf("expected at least 5000 bytes of data, got %d", totalLen)
	}

	t.Logf("Test passed: received %d bytes in %d messages", totalLen, len(messages))
}
