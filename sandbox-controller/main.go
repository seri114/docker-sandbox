package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/seri114/docker-sandbox/controller/config"
	"github.com/seri114/docker-sandbox/controller/internal/docker"
	"github.com/seri114/docker-sandbox/controller/internal/handler"
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

	ctx, cancel := s.dockerClient.Context()
	defer cancel()

	if err := s.dockerClient.StartContainer(ctx, req.ContainerID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to start container: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(handler.StartContainerResponse{Status: "running"})
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

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a context for logs streaming that can be cancelled
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Get logs with follow=true for streaming
	logReader, err := s.dockerClient.ContainerLogs(ctx, containerID, true)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get logs: %s", err), http.StatusInternalServerError)
		return
	}
	defer logReader.Close()

	// Create logs streamer
	streamer := handler.NewLogsStreamer(logReader)
	logChan := make(chan handler.LogMessage, 10)

	// Start streaming in goroutine
	go streamer.StreamTo(logChan)

	// Flush the response writer periodically
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send SSE events
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
		case <-time.After(30 * time.Second):
			// Send a keep-alive comment every 30 seconds
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func main() {
	cfg := config.Default()

	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %s", err)
	}
	defer dockerClient.Close()

	server := NewServer(dockerClient)

	log.Println("Starting sandbox controller on :8080")
	if err := http.ListenAndServe(":8080", server.router); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}
