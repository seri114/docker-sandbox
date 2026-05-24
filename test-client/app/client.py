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

    def create_container(
        self, image: str = "python:3.12-alpine", code: str = ""
    ) -> str:
        """Create a new container.

        Args:
            image: Docker image to use for the container
            code: Python code to execute (optional, defaults to "print('Ready')")

        Returns:
            container_id: The ID of the created container
        """
        response = self.client.post(
            f"{self.base_url}/containers/create", json={"image": image, "code": code}
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
            json={"container_id": container_id, "code": code, "timeout": timeout},
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
            f"{self.base_url}/containers/stop", json={"container_id": container_id}
        )
        return response.json()

    def stream_logs(self, container_id: str, timeout: float = 30.0):
        """Stream logs from a container.

        Args:
            container_id: The ID of the container to get logs from
            timeout: Request timeout in seconds

        Returns:
            Streaming response object usable as context manager
        """
        return self.client.stream(
            "GET",
            f"{self.base_url}/containers/logs",
            params={"id": container_id},
            timeout=timeout,
        )

    def read_stream_lines(self, container_id: str, timeout: float = 30.0):
        """Read streaming logs line by line.

        Args:
            container_id: The ID of the container to get logs from
            timeout: Request timeout in seconds

        Yields:
            Log lines as strings
        """
        with self.stream_logs(container_id, timeout) as response:
            if response.status_code == 200:
                for line in response.iter_lines():
                    yield line

    def close(self):
        """Close the HTTP client."""
        self.client.close()


class AsyncSandboxClient:
    """Async HTTP client for communicating with sandbox-controller."""

    def __init__(self, base_url: str):
        """Initialize the AsyncSandboxClient.

        Args:
            base_url: Base URL of the sandbox-controller API
        """
        self.base_url = base_url

    async def create_container(
        self, image: str = "python:3.12-alpine", code: str = ""
    ) -> str:
        """Create a new container."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            response = await client.post(
                f"{self.base_url}/containers/create",
                json={"image": image, "code": code},
            )
            data = response.json()
            return data["container_id"]

    async def stream_logs(self, container_id: str, timeout: float = 30.0):
        """Stream logs from a container.

        Args:
            container_id: The ID of the container to get logs from
            timeout: Request timeout in seconds

        Returns:
            Tuple of (response, client) for use with async context manager
        """
        client = httpx.AsyncClient(timeout=timeout)
        try:
            response = await client.stream(
                "GET",
                f"{self.base_url}/containers/logs",
                params={"id": container_id},
                timeout=timeout,
            )
            return response, client
        except:
            await client.aclose()
            raise

    async def read_stream_lines(self, container_id: str, timeout: float = 30.0):
        """Read streaming logs line by line asynchronously.

        Args:
            container_id: The ID of the container to get logs from
            timeout: Request timeout in seconds

        Yields:
            Log lines as strings
        """
        response, client = await self.stream_logs(container_id, timeout)
        try:
            if response.status_code == 200:
                async for line in response.aiter_lines():
                    yield line
        finally:
            await client.aclose()

    async def aclose(self):
        """Close resources (no-op for this stateless client)."""
        pass
