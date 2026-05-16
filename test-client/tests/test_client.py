"""Tests for SandboxClient."""

import pytest
from unittest.mock import Mock, patch, MagicMock
from app.client import SandboxClient


@pytest.fixture
def mock_httpx_client():
    """Mock httpx.Client."""
    with patch('app.client.httpx.Client') as mock:
        yield mock


@pytest.fixture
def client(mock_httpx_client):
    """Create a SandboxClient instance with mocked httpx client."""
    mock_client_instance = Mock()
    mock_httpx_client.return_value = mock_client_instance
    return SandboxClient("http://localhost:8000")


def test_client_init(client, mock_httpx_client):
    """Test SandboxClient initialization."""
    # Verify httpx.Client was called with 30s timeout
    mock_httpx_client.assert_called_once_with(timeout=30.0)

    # Verify base_url is stored
    assert client.base_url == "http://localhost:8000"

    # Verify client attribute exists
    assert client.client is not None


def test_client_init_custom_base_url():
    """Test SandboxClient with custom base URL."""
    with patch('app.client.httpx.Client') as mock:
        mock_client_instance = Mock()
        mock.return_value = mock_client_instance

        client = SandboxClient("http://custom:9000")

        assert client.base_url == "http://custom:9000"
        mock.assert_called_once_with(timeout=30.0)


def test_create_container(client):
    """Test create_container method."""
    # Mock the response
    mock_response = Mock()
    mock_response.json.return_value = {"container_id": "test-container-123"}
    client.client.post.return_value = mock_response

    # Call method
    result = client.create_container(image="python:3.12-alpine")

    # Verify request
    client.client.post.assert_called_once_with(
        "http://localhost:8000/containers/create",
        json={"image": "python:3.12-alpine"}
    )

    # Verify result
    assert result == "test-container-123"


def test_create_container_default_image(client):
    """Test create_container with default image."""
    mock_response = Mock()
    mock_response.json.return_value = {"container_id": "container-456"}
    client.client.post.return_value = mock_response

    result = client.create_container()

    client.client.post.assert_called_once_with(
        "http://localhost:8000/containers/create",
        json={"image": "python:3.12-alpine"}
    )
    assert result == "container-456"


def test_start_container(client):
    """Test start_container method."""
    mock_response = Mock()
    mock_response.json.return_value = {"status": "started"}
    client.client.post.return_value = mock_response

    result = client.start_container("container-123", code="print('hello')")

    client.client.post.assert_called_once_with(
        "http://localhost:8000/containers/start",
        json={"container_id": "container-123", "code": "print('hello')", "timeout": 30}
    )
    assert result == {"status": "started"}


def test_start_container_custom_timeout(client):
    """Test start_container with custom timeout."""
    mock_response = Mock()
    mock_response.json.return_value = {"status": "started"}
    client.client.post.return_value = mock_response

    result = client.start_container("container-123", code="x = 1", timeout=60)

    client.client.post.assert_called_once_with(
        "http://localhost:8000/containers/start",
        json={"container_id": "container-123", "code": "x = 1", "timeout": 60}
    )
    assert result == {"status": "started"}


def test_stop_container(client):
    """Test stop_container method."""
    mock_response = Mock()
    mock_response.json.return_value = {"status": "stopped"}
    client.client.post.return_value = mock_response

    result = client.stop_container("container-123")

    client.client.post.assert_called_once_with(
        "http://localhost:8000/containers/stop",
        json={"container_id": "container-123"}
    )
    assert result == {"status": "stopped"}


def test_stream_logs(client):
    """Test stream_logs method returns streaming response."""
    mock_response = Mock()
    mock_response.status_code = 200
    client.client.stream.return_value = mock_response

    result = client.stream_logs("container-123")

    client.client.stream.assert_called_once_with(
        "GET",
        "http://localhost:8000/containers/logs",
        params={"container_id": "container-123"}
    )
    assert result == mock_response


def test_close(client):
    """Test close method closes httpx client."""
    client.close()

    client.client.close.assert_called_once()
