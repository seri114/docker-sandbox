package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/seri114/docker-sandbox/config"
	"github.com/seri114/docker-sandbox/internal/docker"
	"github.com/seri114/docker-sandbox/internal/handler"
)

// Server represents the HTTP server for the sandbox controller.
type Server struct {
	dockerClient *docker.DockerClient
	router       *mux.Router
}

// NewServer creates a new Server instance.
func NewServer(dockerClient *docker.DockerClient) *Server {
	s := &Server{
		dockerClient: dockerClient,
		router:       mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures the HTTP endpoints.
func (s *Server) setupRoutes() {
	s.router.HandleFunc("/containers/create", s.handleCreate).Methods("POST")
	s.router.HandleFunc("/containers/start", s.handleStart).Methods("POST")
	s.router.HandleFunc("/containers/stop", s.handleStop).Methods("POST")
	s.router.HandleFunc("/containers/logs", s.handleLogs).Methods("GET")
}

// handleCreate handles the container creation endpoint.
func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req handler.CreateContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %s", err), http.StatusBadRequest)
		return
	}

	// Set default values if not provided
	if req.Code == "" {
		req.Code = "print('Ready')"
	}
	if req.Memory == 0 {
		req.Memory = 128 * 1024 * 1024 // 128MB in bytes
	}
	if req.CPU == 0 {
		req.CPU = 0.5 // 0.5 CPU
	}

	// Validate request
	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %s", err), http.StatusBadRequest)
		return
	}

	ctx, cancel := s.dockerClient.Context()
	defer cancel()

	containerConfig := req.ToContainerConfig()
	hostConfig := req.ToHostConfig()

	containerID, err := s.dockerClient.CreateContainer(ctx, containerConfig, hostConfig)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create container: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handler.CreateContainerResponse{ContainerID: containerID})
}

// handleStart handles the container start endpoint.
// NOTE: Start is executed asynchronously due to Docker Desktop for Mac socket proxy bug.
// The container start API may not respond, but the container executes normally.
func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	var req handler.StartContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %s", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %s", err), http.StatusBadRequest)
		return
	}

	log.Printf("[DEBUG] Starting container (async): %s", req.ContainerID)

	// Execute container start asynchronously to avoid socket proxy timeout
	go func() {
		ctx, cancel := s.dockerClient.Context()
		defer cancel()

		if err := s.dockerClient.StartContainer(ctx, req.ContainerID); err != nil {
			log.Printf("[ERROR] Failed to start container %s: %v", req.ContainerID, err)
		} else {
			log.Printf("[DEBUG] Container started successfully: %s", req.ContainerID)
		}
	}()

	// Return immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handler.StartContainerResponse{Status: "starting"})
}

// handleStop handles the container stop endpoint.
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	var req handler.StopContainerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %s", err), http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %s", err), http.StatusBadRequest)
		return
	}

	ctx, cancel := s.dockerClient.Context()
	defer cancel()

	if err := s.dockerClient.StopAndRemoveContainer(ctx, req.ContainerID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop container: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handler.StopContainerResponse{Status: "stopped"})
}

// handleLogs handles the container logs streaming endpoint with SSE.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		http.Error(w, "Missing container id parameter", http.StatusBadRequest)
		return
	}

	log.Printf("[DEBUG] Streaming logs for container: %s", containerID)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a context for logs streaming that can be cancelled
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	// Wait for container to start and be running (max 5 seconds)
	for i := 0; i < 10; i++ {
		cont, err := s.dockerClient.InspectContainer(ctx, containerID)
		if err == nil && cont.State.Running {
			log.Printf("[DEBUG] Container %s is running", containerID)
			break
		}
		if i < 9 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Get logs with follow=true for streaming
	logReader, err := s.dockerClient.ContainerLogs(ctx, containerID, true)
	if err != nil {
		log.Printf("[ERROR] Failed to get logs: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get logs: %s", err), http.StatusInternalServerError)
		return
	}
	defer logReader.Close()

	log.Printf("[DEBUG] Log reader created for container: %s", containerID)

	// Create logs streamer
	streamer := handler.NewLogsStreamer(logReader)
	logChan := make(chan handler.LogMessage, 10)

	// Start streaming in goroutine
	go streamer.StreamTo(logChan)

	log.Printf("[DEBUG] Starting SSE stream for container: %s", containerID)

	// Flush the response writer periodically
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	messageCount := 0
	emptyCount := 0

	// Send SSE events
	for {
		select {
		case msg, ok := <-logChan:
			if !ok {
				log.Printf("[DEBUG] Log channel closed for container: %s, messages sent: %d", containerID, messageCount)
				return
			}
			messageCount++
			emptyCount = 0
			fmt.Fprintf(w, "data: %s\n\n", msg.ToJSON())
			flusher.Flush()
		case <-ctx.Done():
			log.Printf("[DEBUG] Context cancelled for container: %s, messages sent: %d", containerID, messageCount)
			return
		case <-time.After(1 * time.Second):
			// Check if container is still running
			cont, err := s.dockerClient.InspectContainer(ctx, containerID)
			if err == nil && !cont.State.Running && cont.State.ExitCode == 0 {
				// Container finished successfully, check if we got any logs
				if messageCount == 0 {
					// No logs received, container might have finished too quickly
					log.Printf("[DEBUG] Container finished with no logs, sending completion message")
					fmt.Fprintf(w, "data: {\"stream\":\"stdout\",\"data\":\"(no output)\"}\n\n")
					flusher.Flush()
				}
				return
			}
			emptyCount++
			if emptyCount > 30 {
				// No logs for 30 seconds, close stream
				log.Printf("[DEBUG] No logs for 30 seconds, closing stream")
				return
			}
		}
	}
}

func main() {
	cfg := config.Default()

	// Override DockerHost from environment variable if set
	if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost != "" {
		cfg.DockerHost = dockerHost
	}

	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %s", err)
	}
	defer dockerClient.Close()

	server := NewServer(dockerClient)

	// Setup CORS with environment variable support
	allowedOrigins := os.Getenv("CORS_ORIGINS")
	var origins []string
	if allowedOrigins != "" {
		origins = strings.Split(allowedOrigins, ",")
	} else {
		// Default to localhost only for security
		origins = []string{
			"http://localhost:8080",
			"http://localhost:18080",
			"http://127.0.0.1:8080",
			"http://127.0.0.1:18080",
		}
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
	})

	handler := c.Handler(server.router)

	log.Println("Starting sandbox controller on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}
