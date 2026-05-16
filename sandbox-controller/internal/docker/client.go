package docker

import (
	"context"
	"time"

	"github.com/docker/docker/client"
)

type DockerClient struct {
	cli     *client.Client
	timeout time.Duration
}

func NewClient(host string, timeout time.Duration) *DockerClient {
	cli, err := client.NewClientWithOpts(client.WithHost(host))
	if err != nil {
		panic(err)
	}
	return &DockerClient{
		cli:     cli,
		timeout: timeout,
	}
}

func (d *DockerClient) Close() error {
	return d.cli.Close()
}

func (d *DockerClient) Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d.timeout)
}
