"""HTTP client for sandbox-controller."""

import httpx


class SandboxClient:
    """HTTP client for communicating with sandbox-controller."""

    def __init__(self, base_url: str):
        """Initialize the SandboxClient.

        Args:
            base_url: Base URL of the sandbox-controller API (e.g., "http://localhost:8000")
        """
        self.base_url = base_url
        self.client = httpx.Client(timeout=30.0)

    def create_container(self, image: str = "python:3.12-alpine") -> str:
        """Create a new container.

        Args:
            image: Docker image to use for the container

        Returns:
            container_id: The ID of the created container
        """
        response = self.client.post(
            f"{self.base_url}/containers/create",
            json={"image": image}
        )
        data = response.json()
        return data["container_id"]

    def start_container(self, container_id: str, code: str, timeout: int = 30) -> dict:
        """Start a container with code to execute.

        Args:
            container_id: The ID of the container to start
            code: Python code to execute in the container
            timeout: Execution timeout in seconds

        Returns:
            Response data from the server
        """
        response = self.client.post(
            f"{self.base_url}/containers/start",
            json={"container_id": container_id, "code": code, "timeout": timeout}
        )
        return response.json()

    def stop_container(self, container_id: str) -> dict:
        """Stop a running container.

        Args:
            container_id: The ID of the container to stop

        Returns:
            Response data from the server
        """
        response = self.client.post(
            f"{self.base_url}/containers/stop",
            json={"container_id": container_id}
        )
        return response.json()

    def stream_logs(self, container_id: str):
        """Stream logs from a container.

        Args:
            container_id: The ID of the container to get logs from

        Returns:
            Streaming response object
        """
        return self.client.stream(
            "GET",
            f"{self.base_url}/containers/logs",
            params={"container_id": container_id}
        )

    def close(self):
        """Close the HTTP client."""
        self.client.close()
