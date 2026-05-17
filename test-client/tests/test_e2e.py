"""E2E tests for sandbox test client."""

import asyncio

import httpx
import pytest


async def test_e2e_code_execution_with_real_docker():
    """Test complete flow with real Docker container."""
    controller_url = "http://sandbox-controller:8080"

    try:
        async with httpx.AsyncClient(timeout=30.0) as client:
            code = """print("Hello")
print("World")
for i in range(3):
    print(f"Count: {i}")"""

            # Create container
            response = await client.post(
                f"{controller_url}/containers/create",
                json={"image": "python:3.12-alpine", "code": code},
            )
            assert response.status_code == 200
            container_id = response.json()["container_id"]

            # Start container
            await client.post(
                f"{controller_url}/containers/start",
                json={"container_id": container_id, "code": code, "timeout": 30},
            )

            await asyncio.sleep(0.5)

            # Stream logs
            async with client.stream(
                "GET", f"{controller_url}/containers/logs", params={"id": container_id}
            ) as response:
                assert response.status_code == 200
                assert response.headers["content-type"] == "text/event-stream"

                output_lines = []
                async for line in response.aiter_lines():
                    if line.startswith("data: "):
                        import json

                        json_str = line[6:]
                        try:
                            msg = json.loads(json_str)
                            if "data" in msg and msg["data"]:
                                output_lines.append(msg["data"].strip())
                        except Exception:
                            pass

            assert len(output_lines) >= 3
            assert any("Hello" in line for line in output_lines)

            # Cleanup
            await client.post(
                f"{controller_url}/containers/stop", json={"container_id": container_id}
            )

    except httpx.ConnectError:
        pytest.skip("Controller not available")


async def test_e2e_sse_format():
    """Test SSE format is correct."""
    controller_url = "http://sandbox-controller:8080"

    try:
        async with httpx.AsyncClient(timeout=30.0) as client:
            code = 'print("Test")'

            response = await client.post(
                f"{controller_url}/containers/create",
                json={"image": "python:3.12-alpine", "code": code},
            )
            container_id = response.json()["container_id"]

            await client.post(
                f"{controller_url}/containers/start",
                json={"container_id": container_id, "code": code, "timeout": 30},
            )

            await asyncio.sleep(0.5)

            # Verify SSE format
            async with client.stream(
                "GET", f"{controller_url}/containers/logs", params={"id": container_id}
            ) as response:
                raw_data = b""
                async for chunk in response.aiter_bytes():
                    raw_data += chunk
                    if len(raw_data) > 100:
                        break

            data_str = raw_data.decode("utf-8", errors="ignore")
            assert "data: " in data_str
            assert "\n\n" in data_str  # SSE requires double newline

            # Cleanup
            await client.post(
                f"{controller_url}/containers/stop", json={"container_id": container_id}
            )

    except httpx.ConnectError:
        pytest.skip("Controller not available")


async def test_e2e_cancel_execution():
    """Test cancelling a running execution."""
    controller_url = "http://sandbox-controller:8080"

    try:
        async with httpx.AsyncClient(timeout=30.0) as client:
            # Long running code
            code = """import time
print("Starting...")
for i in range(10):
    print(f"Step {i}")
    time.sleep(0.5)
print("Done")"""

            # Create container
            response = await client.post(
                f"{controller_url}/containers/create",
                json={"image": "python:3.12-alpine", "code": code},
            )
            assert response.status_code == 200
            container_id = response.json()["container_id"]

            # Start container
            await client.post(
                f"{controller_url}/containers/start",
                json={"container_id": container_id, "code": code, "timeout": 30},
            )

            # Wait for container to start
            await asyncio.sleep(1)

            # Stop the container
            response = await client.post(
                f"{controller_url}/containers/stop", json={"container_id": container_id}
            )
            assert response.status_code == 200

            # Verify container is stopped (inspect should fail)
            try:
                response = await client.get(
                    f"{controller_url}/containers/{container_id}/json"
                )
                # If we get here, container still exists - check if it's stopped/exited
                if response.status_code == 200:
                    data = response.json()
                    assert data["State"]["Running"] is False, (
                        "Container should be stopped"
                    )
            except httpx.HTTPStatusError as e:
                # 404 is ok - container was removed
                assert e.response.status_code == 404

    except httpx.ConnectError:
        pytest.skip("Controller not available")
