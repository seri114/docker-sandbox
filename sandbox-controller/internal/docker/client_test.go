package docker

import (
	"testing"
	"time"

	"github.com/seri114/docker-sandbox/controller/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		DockerHost:     "unix:///var/run/docker.sock",
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
