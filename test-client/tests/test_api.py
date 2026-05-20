"""Tests for FastAPI application."""

import uuid
from unittest.mock import Mock

import pytest
from app.api import ExecuteRequest, app, get_sandbox_client
from fastapi.testclient import TestClient


@pytest.fixture
def mock_sandbox_client():
    """Mock SandboxClient."""
    mock = Mock()
    mock.create_container.return_value = "test-container-123"
    mock.start_container.return_value = {"status": "running"}
    return mock


@pytest.fixture
def client(mock_sandbox_client):
    """Create test client with mocked SandboxClient."""
    app.dependency_overrides[get_sandbox_client] = lambda: mock_sandbox_client
    with TestClient(app) as test_client:
        yield test_client
    app.dependency_overrides.clear()


def test_execute_request_model():
    """Test ExecuteRequest pydantic model."""
    req = ExecuteRequest(code="print('hello')")
    assert req.code == "print('hello')"
    assert req.image == "python:3.12-alpine"
    assert req.timeout == 30


def test_execute_request_custom_values():
    """Test ExecuteRequest with custom values."""
    req = ExecuteRequest(code="x = 1", image="python:3.12-alpine", timeout=60)
    assert req.code == "x = 1"
    assert req.image == "python:3.12-alpine"
    assert req.timeout == 60


def test_create_execution(client, mock_sandbox_client):
    """Test POST /api/execute creates execution."""
    response = client.post("/api/execute", json={"code": "print('hello')"})

    assert response.status_code == 200

    # Verify execution_id is a valid UUID
    data = response.json()
    assert "execution_id" in data
    uuid.UUID(data["execution_id"])  # Will raise if invalid

    # Verify container was created and started
    mock_sandbox_client.create_container.assert_called_once_with(
        image="python:3.12-alpine", code="print('hello')"
    )
    mock_sandbox_client.start_container.assert_called_once_with(
        container_id="test-container-123", code="print('hello')", timeout=30
    )


def test_create_execution_custom_params(client, mock_sandbox_client):
    """Test POST /api/execute with custom parameters."""
    mock_sandbox_client.create_container.return_value = "custom-container-456"

    response = client.post(
        "/api/execute",
        json={
            "code": "import time; time.sleep(1)",
            "image": "python:3.12-alpine",
            "timeout": 60,
        },
    )

    assert response.status_code == 200

    data = response.json()
    assert "execution_id" in data
    uuid.UUID(data["execution_id"])

    mock_sandbox_client.create_container.assert_called_once_with(
        image="python:3.12-alpine", code="import time; time.sleep(1)"
    )
    mock_sandbox_client.start_container.assert_called_once_with(
        container_id="custom-container-456",
        code="import time; time.sleep(1)",
        timeout=60,
    )


def test_create_execution_missing_code(client):
    """Test POST /api/execute without code returns validation error."""
    response = client.post("/api/execute", json={})

    assert response.status_code == 422


def test_app_title():
    """Test FastAPI app has correct title."""
    assert app.title == "Sandbox Test Client"
