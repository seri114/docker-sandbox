package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/mux"

	"github.com/seri114/docker-sandbox/config"
	"github.com/seri114/docker-sandbox/internal/docker"
)

// getDockerHost returns the Docker socket path
func getDockerHost() string {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return host
	}
	// Try common Docker socket paths
	paths := []string{
		"/var/run/docker.sock",
		os.ExpandEnv("/Users/$USER/.docker/run/docker.sock"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return "unix://" + path
		}
	}
	return "unix:///var/run/docker.sock"
}

// TestE2ECodeExecution tests the complete flow: create container, start, stream logs.
func TestE2ECodeExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	cfg := &config.Config{
		DockerHost:     getDockerHost(),
		RequestTimeout: 30 * time.Second,
	}

	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer dockerClient.Close()

	// Create test server
	server := httptest.NewServer(setupTestRouter(dockerClient))
	defer server.Close()

	// Test code
	code := `print("Hello")
print("World")
for i in range(3):
    print(f"Count: {i}")`

	// Step 1: Create container
	createReq := CreateContainerRequest{
		Image:  "python:3.12-alpine",
		Code:   code,
		Memory: 128 * 1024 * 1024,
		CPU:    0.5,
	}
	createBody, _ := json.Marshal(createReq)

	resp, err := http.Post(server.URL+"/containers/create", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("Create request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Create failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createResp CreateContainerResponse
	json.NewDecoder(resp.Body).Decode(&createResp)
	resp.Body.Close()

	if createResp.ContainerID == "" {
		t.Fatal("expected non-empty container ID")
	}

	containerID := createResp.ContainerID
	t.Logf("Created container: %s", containerID)

	// Step 2: Start container
	startReq := StartContainerRequest{
		ContainerID: containerID,
		Code:        code,
		Timeout:     30,
	}
	startBody, _ := json.Marshal(startReq)

	resp, err = http.Post(server.URL+"/containers/start", "application/json", bytes.NewReader(startBody))
	if err != nil {
		t.Fatalf("Start request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Start failed with status %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Wait a bit for container to start
	time.Sleep(500 * time.Millisecond)

	// Step 3: Stream logs
	logURL := server.URL + "/containers/logs?id=" + containerID
	req, _ := http.NewRequest("GET", logURL, nil)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Logs request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify SSE headers
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}

	// Read SSE stream
	var outputLines []string
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			data := string(buf[:n])
			t.Logf("Received: %q", data)

			// Parse SSE format "data: {...}\n\n"
			lines := strings.Split(data, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "data: ") {
					jsonStr := strings.TrimPrefix(line, "data: ")
					var msg LogMessage
					if err := json.Unmarshal([]byte(jsonStr), &msg); err == nil {
						if msg.Data != "" {
							outputLines = append(outputLines, strings.TrimSuffix(msg.Data, "\n"))
						}
					}
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			t.Logf("Read error (may be expected): %v", err)
			break
		}
	}

	t.Logf("Output lines: %v", outputLines)

	// Verify output
	if len(outputLines) < 3 {
		t.Errorf("expected at least 3 output lines, got %d", len(outputLines))
	}

	// Check for expected content
	expectedContent := []string{"Hello", "World", "Count: 0", "Count: 1", "Count: 2"}
	found := make(map[string]bool)
	for _, line := range outputLines {
		for _, expected := range expectedContent {
			if strings.Contains(line, expected) {
				found[expected] = true
			}
		}
	}

	for _, expected := range expectedContent {
		if !found[expected] {
			t.Errorf("expected output to contain %q", expected)
		}
	}

	// Step 4: Stop and remove container
	stopReq := StopContainerRequest{ContainerID: containerID}
	stopBody, _ := json.Marshal(stopReq)

	resp, err = http.Post(server.URL+"/containers/stop", "application/json", bytes.NewReader(stopBody))
	if err != nil {
		t.Logf("Stop request failed (may be ok): %v", err)
	} else {
		resp.Body.Close()
	}
}

// TestE2EStreamingWithDelay tests that streaming works with delayed output.
func TestE2EStreamingWithDelay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	cfg := &config.Config{
		DockerHost:     getDockerHost(),
		RequestTimeout: 30 * time.Second,
	}

	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer dockerClient.Close()

	server := httptest.NewServer(setupTestRouter(dockerClient))
	defer server.Close()

	// Code with delays
	code := `import time
for i in range(3):
    print(f"Line {i}")
    time.sleep(0.5)`

	createReq := CreateContainerRequest{
		Image:  "python:3.12-alpine",
		Code:   code,
		Memory: 128 * 1024 * 1024,
		CPU:    0.5,
	}
	createBody, _ := json.Marshal(createReq)

	resp, err := http.Post(server.URL+"/containers/create", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("Create request failed: %v", err)
	}

	var createResp CreateContainerResponse
	json.NewDecoder(resp.Body).Decode(&createResp)
	resp.Body.Close()

	containerID := createResp.ContainerID

	// Start container
	startReq := StartContainerRequest{
		ContainerID: containerID,
		Code:        code,
		Timeout:     30,
	}
	startBody, _ := json.Marshal(startReq)

	resp, err = http.Post(server.URL+"/containers/start", "application/json", bytes.NewReader(startBody))
	if err != nil {
		t.Fatalf("Start request failed: %v", err)
	}
	resp.Body.Close()

	time.Sleep(500 * time.Millisecond)

	// Stream logs and measure timing
	logURL := server.URL + "/containers/logs?id=" + containerID
	req, _ := http.NewRequest("GET", logURL, nil)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Logs request failed: %v", err)
	}
	defer resp.Body.Close()

	startTime := time.Now()
	messageCount := 0
	lastMessageTime := startTime

	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			lastMessageTime = time.Now()
			data := string(buf[:n])
			if strings.Contains(data, "data:") {
				messageCount++
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			break
		}
		// Timeout if no message for 3 seconds
		if time.Since(lastMessageTime) > 3*time.Second {
			break
		}
	}

	t.Logf("Received %d messages over %v", messageCount, time.Since(startTime))

	// We expect at least some messages
	if messageCount < 1 {
		t.Error("expected at least 1 message from delayed output")
	}

	// Clean up
	stopReq := StopContainerRequest{ContainerID: containerID}
	stopBody, _ := json.Marshal(stopReq)
	http.Post(server.URL+"/containers/stop", "application/json", bytes.NewReader(stopBody))
}

// setupTestRouter creates a test router with all handlers.
func setupTestRouter(dockerClient *docker.DockerClient) *mux.Router {
	router := mux.NewRouter()

	// Handler functions adapted for testing
	router.HandleFunc("/containers/create", func(w http.ResponseWriter, r *http.Request) {
		var req CreateContainerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Code == "" {
			req.Code = "print('Ready')"
		}
		if req.Memory == 0 {
			req.Memory = 128 * 1024 * 1024
		}
		if req.CPU == 0 {
			req.CPU = 0.5
		}

		if err := req.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := dockerClient.Context()
		defer cancel()

		containerConfig := req.ToContainerConfig()
		hostConfig := req.ToHostConfig()

		containerID, err := dockerClient.CreateContainer(ctx, containerConfig, hostConfig)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CreateContainerResponse{ContainerID: containerID})
	}).Methods("POST")

	router.HandleFunc("/containers/start", func(w http.ResponseWriter, r *http.Request) {
		var req StartContainerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := req.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := dockerClient.Context()
		defer cancel()

		if err := dockerClient.StartContainer(ctx, req.ContainerID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(StartContainerResponse{Status: "started"})
	}).Methods("POST")

	router.HandleFunc("/containers/stop", func(w http.ResponseWriter, r *http.Request) {
		var req StopContainerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := req.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := dockerClient.Context()
		defer cancel()

		if err := dockerClient.StopAndRemoveContainer(ctx, req.ContainerID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(StopContainerResponse{Status: "stopped"})
	}).Methods("POST")

	router.HandleFunc("/containers/logs", func(w http.ResponseWriter, r *http.Request) {
		containerID := r.URL.Query().Get("id")
		if containerID == "" {
			http.Error(w, "Missing container id parameter", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		logReader, err := dockerClient.ContainerLogs(ctx, containerID, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer logReader.Close()

		streamer := NewLogsStreamer(logReader)
		logChan := make(chan LogMessage, 10)

		go streamer.StreamTo(logChan)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		for {
			select {
			case msg, ok := <-logChan:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", msg.ToJSON())
				flusher.Flush()
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				cont, err := dockerClient.InspectContainer(ctx, containerID)
				if err == nil && !cont.State.Running {
					return
				}
			}
		}
	}).Methods("GET")

	return router
}

// TestE2ECancelExecution tests cancelling a running execution.
func TestE2ECancelExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	cfg := &config.Config{
		DockerHost:     getDockerHost(),
		RequestTimeout: 30 * time.Second,
	}

	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer dockerClient.Close()

	// Test Docker connection by trying to list containers
	ctx, cancel := dockerClient.Context()
	defer cancel()
	_, err = dockerClient.Client().ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		t.Skipf("Cannot connect to Docker daemon: %v", err)
	}

	server := httptest.NewServer(setupTestRouter(dockerClient))
	defer server.Close()

	// Code with long delay
	code := `import time
print("Starting...")
for i in range(10):
    print(f"Step {i}")
    time.sleep(1)
print("Done")`

	// Create and start container
	createReq := CreateContainerRequest{
		Image:  "python:3.12-alpine",
		Code:   code,
		Memory: 128 * 1024 * 1024,
		CPU:    0.5,
	}
	createBody, _ := json.Marshal(createReq)

	resp, err := http.Post(server.URL+"/containers/create", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("Create request failed: %v", err)
	}

	// Read response body for debugging
	bodyBytes, _ := io.ReadAll(resp.Body)
	t.Logf("Create response status: %d, body: %s", resp.StatusCode, string(bodyBytes))

	var createResp CreateContainerResponse
	if err := json.Unmarshal(bodyBytes, &createResp); err != nil {
		t.Fatalf("Failed to decode create response: %v, body: %s", err, string(bodyBytes))
	}

	containerID := createResp.ContainerID
	t.Logf("Created container ID: %q", containerID)

	if containerID == "" {
		t.Fatal("received empty container ID from create response")
	}

	// Start container
	startReq := StartContainerRequest{
		ContainerID: containerID,
		Code:        code,
		Timeout:     30,
	}
	startBody, _ := json.Marshal(startReq)

	resp, err = http.Post(server.URL+"/containers/start", "application/json", bytes.NewReader(startBody))
	if err != nil {
		t.Fatalf("Start request failed: %v", err)
	}
	resp.Body.Close()

	// Wait a bit for container to start and produce some output
	time.Sleep(2 * time.Second)

	// Cancel execution
	stopReq := StopContainerRequest{ContainerID: containerID}
	stopBody, _ := json.Marshal(stopReq)

	resp, err = http.Post(server.URL+"/containers/stop", "application/json", bytes.NewReader(stopBody))
	if err != nil {
		t.Fatalf("Stop request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Stop failed with status %d: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	// Verify container is stopped and removed
	ctx, cancel = dockerClient.Context()
	defer cancel()

	// Try to inspect - should fail because container is removed
	_, err = dockerClient.InspectContainer(ctx, containerID)
	if err == nil {
		t.Error("expected container to be removed, but it still exists")
	}

	t.Logf("Container %s was successfully cancelled and removed", containerID)
}
