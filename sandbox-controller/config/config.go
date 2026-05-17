package config

import "time"

type Config struct {
	DockerHost     string
	RequestTimeout time.Duration
}

func Default() *Config {
	return &Config{
		DockerHost:     "unix:///var/run/docker.sock",
		RequestTimeout: 60 * time.Second, // Increased for Docker Desktop Mac
	}
}
