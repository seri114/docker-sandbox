package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/container"
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

// CreateContainer creates a new Docker container with the given configuration.
// It returns the container ID and any error that occurred.
func (d *DockerClient) CreateContainer(ctx context.Context, config *container.Config, hostConfig *container.HostConfig) (string, error) {
	resp, err := d.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// StartContainer starts a Docker container with the given ID.
func (d *DockerClient) StartContainer(ctx context.Context, id string) error {
	return d.cli.ContainerStart(ctx, id, container.StartOptions{})
}
