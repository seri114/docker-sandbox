package docker

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("unix:///var/run/docker.sock", 10*time.Second)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.cli == nil {
		t.Fatal("expected docker client to be initialized")
	}
}
