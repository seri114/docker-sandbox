package docker

import (
	"context"
	"time"

	"github.com/docker/docker/client"

	"github.com/seri114/docker-sandbox/controller/config"
)

type DockerClient struct {
	cli     *client.Client
	timeout time.Duration
}

// NewClient creates a new Docker client with the given configuration.
// It returns an error if the client cannot be created.
func NewClient(cfg *config.Config) (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(cfg.DockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	return &DockerClient{
		cli:     cli,
		timeout: cfg.RequestTimeout,
	}, nil
}

// Client returns the underlying Docker client for direct API access.
func (d *DockerClient) Client() *client.Client {
	return d.cli
}

func (d *DockerClient) Close() error {
	return d.cli.Close()
}

func (d *DockerClient) Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d.timeout)
}
