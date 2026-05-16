package docker

import (
	"context"
	"io"
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

// RemoveContainer removes a Docker container with the given ID.
// force forcefully removes the container even if it is running.
func (d *DockerClient) RemoveContainer(ctx context.Context, id string, force bool) error {
	return d.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
}

// ContainerLogs returns the logs from a container.
// follow specifies whether to stream the logs as they are produced.
// It returns an io.ReadCloser that provides Docker JSONLog formatted output.
func (d *DockerClient) ContainerLogs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Timestamps: false,
		Tail:       "",
	}
	return d.cli.ContainerLogs(ctx, id, options)
}

// StopAndRemoveContainer stops and removes a Docker container with the given ID.
// It first attempts to stop the container gracefully with a 10 second timeout,
// then removes it from the system.
func (d *DockerClient) StopAndRemoveContainer(ctx context.Context, id string) error {
	// Stop the container with a 10 second timeout
	timeout := int(10 * time.Second)
	stopErr := d.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})

	// Remove the container (force remove if stop failed)
	return d.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: stopErr != nil})
}
