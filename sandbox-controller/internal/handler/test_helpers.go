package handler

import (
	"github.com/seri114/docker-sandbox/internal/constants"
)

// GetDefaultTestConfig returns a default test configuration for container creation.
// This centralizes test configuration and ensures consistency across tests.
func GetDefaultTestConfig() CreateContainerRequest {
	return CreateContainerRequest{
		Image:  constants.DefaultSandboxImage,
		Code:   "print('hello')",  // Default test code
		Memory: 128 * 1024 * 1024, // 128MB
		CPU:    0.5,               // 50% of a CPU
	}
}
