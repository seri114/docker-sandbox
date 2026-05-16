"""End-to-end tests for SandboxClient.

These tests require the sandbox-controller to be running.
"""

import time
import pytest
from app.client import SandboxClient


@pytest.mark.e2e
def test_full_execution():
    """Test full container lifecycle: create, start, stop, close.

    This is an end-to-end test that requires sandbox-controller to be running.
    """
    # Arrange: Create SandboxClient pointing to the controller
    client = SandboxClient("http://localhost:8000")

    try:
        # Act 1: Create a container
        container_id = client.create_container(image="python:3.12-alpine")

        # Assert 1: Verify we got a container_id
        assert container_id is not None
        assert isinstance(container_id, str)
        assert len(container_id) > 0

        # Act 2: Start the container with test code
        test_code = """
print('Hello from sandbox!')
import sys
print(f'Python version: {sys.version}')
"""
        start_result = client.start_container(container_id, code=test_code)

        # Assert 2: Verify start succeeded
        assert start_result is not None
        assert "status" in start_result

        # Act 3: Wait for execution
        time.sleep(2)

        # Act 4: Stop the container
        stop_result = client.stop_container(container_id)

        # Assert 4: Verify stop succeeded
        assert stop_result is not None
        assert "status" in stop_result

    finally:
        # Cleanup: Always close the client
        client.close()
