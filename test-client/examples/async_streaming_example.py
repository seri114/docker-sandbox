"""Async streaming example with time.sleep for periodic log retrieval."""

import asyncio
import json
import time
from app.client import AsyncSandboxClient


async def async_stream_logs_with_polling(container_id: str, interval: float = 0.5):
    """Stream logs using asyncio.sleep for periodic polling.

    Args:
        container_id: Container ID to stream logs from
        interval: Sleep interval between log fetches (seconds)
    """
    client = AsyncSandboxClient("http://localhost:8000")

    try:
        print(f"Starting async log streaming for container {container_id}...")
        print("Press Ctrl+C to stop\n")

        start_time = time.time()
        timeout = 30  # Maximum streaming duration

        while time.time() - start_time < timeout:
            try:
                # Open streaming connection
                response, http_client = await client.stream_logs(container_id)

                try:
                    if response.status_code == 200:
                        # Read available data
                        async for line in response.aiter_lines():
                            if line.startswith("data: "):
                                json_str = line[6:]
                                try:
                                    msg = json.loads(json_str)
                                    if "data" in msg and msg["data"]:
                                        timestamp = time.strftime("%H:%M:%S")
                                        data = msg["data"].strip()
                                        print(f"[{timestamp}] {data}")
                                except json.JSONDecodeError:
                                    continue
                finally:
                    await http_client.aclose()

                # No more data for now, wait before next poll
                await asyncio.sleep(interval)

            except KeyboardInterrupt:
                print("\n\nStopping log streaming...")
                break
            except Exception as e:
                print(f"Error streaming logs: {e}")
                await asyncio.sleep(interval)

    except Exception as e:
        print(f"Fatal error: {e}")


async def async_main():
    """Main async example demonstrating container execution with log streaming."""
    container_id = None

    try:
        client = AsyncSandboxClient("http://localhost:8000")

        # Example 1: Simple output with delays
        print("=== Async Example: Simple delayed output ===\n")

        code = """import time

print("Starting async execution...")
for i in range(5):
    print(f"Processing step {i+1}/5")
    time.sleep(1)
print("Execution complete!")
"""

        # Create container
        container_id = await client.create_container(
            image="python:3.12-alpine", code=code
        )
        print(f"Created container: {container_id}\n")

        # Start container (using httpx directly since start_container
        # doesn't exist in async version)
        async with httpx.AsyncClient(timeout=30.0) as http_client:
            await http_client.post(
                "http://localhost:8000/containers/start",
                json={"container_id": container_id, "code": code, "timeout": 30},
            )

        print("Container started, beginning log streaming...\n")

        # Stream logs with polling
        await asyncio.sleep(0.5)  # Initial delay to let container start
        await async_stream_logs_with_polling(container_id, interval=0.3)

        # Cleanup
        print("\nStopping container...")
        async with httpx.AsyncClient(timeout=30.0) as http_client:
            await http_client.post(
                "http://localhost:8000/containers/stop",
                json={"container_id": container_id},
            )
        print("Done!")

    except Exception as e:
        print(f"Error: {e}")
        if container_id:
            try:
                async with httpx.AsyncClient(timeout=30.0) as http_client:
                    await http_client.post(
                        "http://localhost:8000/containers/stop",
                        json={"container_id": container_id},
                    )
            except Exception:
                pass


async def async_example_multiple_containers():
    """Example: Monitor multiple containers concurrently."""
    print("\n=== Async Example: Multiple containers ===\n")

    tasks = []
    container_ids = []

    # Create and start multiple containers
    codes = [
        "import time\nprint('Container A starting'); "
        "time.sleep(2); print('Container A done')",
        "import time\nprint('Container B starting'); "
        "time.sleep(3); print('Container B done')",
        "import time\nprint('Container C starting'); "
        "time.sleep(1); print('Container C done')",
    ]

    try:
        client = AsyncSandboxClient("http://localhost:8000")

        # Create containers
        for i, code in enumerate(codes):
            container_id = await client.create_container(
                image="python:3.12-alpine", code=code
            )
            container_ids.append(container_id)
            print(f"Created container {i}: {container_id}")

        # Start containers
        async with httpx.AsyncClient(timeout=30.0) as http_client:
            for container_id in container_ids:
                await http_client.post(
                    "http://localhost:8000/containers/start",
                    json={
                        "container_id": container_id,
                        "code": codes[container_ids.index(container_id)],
                        "timeout": 30,
                    },
                )

        print("\nAll containers started, streaming logs...\n")

        # Stream logs from all containers concurrently
        async def monitor_container(idx, container_id):
            print(f"[Container {idx}] Starting monitor...")
            response, http_client = await client.stream_logs(container_id)

            try:
                async for line in response.aiter_lines():
                    if line.startswith("data: "):
                        json_str = line[6:]
                        try:
                            msg = json.loads(json_str)
                            if "data" in msg and msg["data"]:
                                print(f"[Container {idx}] {msg['data'].strip()}")
                        except json.JSONDecodeError:
                            continue
            finally:
                await http_client.aclose()
                print(f"[Container {idx}] Monitor ended")

        # Start all monitoring tasks
        for i, container_id in enumerate(container_ids):
            tasks.append(asyncio.create_task(monitor_container(i, container_id)))

        # Wait for all tasks to complete
        await asyncio.gather(*tasks)

        # Cleanup
        print("\nStopping all containers...")
        async with httpx.AsyncClient(timeout=30.0) as http_client:
            for container_id in container_ids:
                await http_client.post(
                    "http://localhost:8000/containers/stop",
                    json={"container_id": container_id},
                )

        print("All containers stopped!")

    except Exception as e:
        print(f"Error: {e}")
        if container_ids:
            try:
                async with httpx.AsyncClient(timeout=30.0) as http_client:
                    for container_id in container_ids:
                        await http_client.post(
                            "http://localhost:8000/containers/stop",
                            json={"container_id": container_id},
                        )
            except Exception:
                pass


if __name__ == "__main__":
    import httpx

    print("Docker Sandbox - Async Streaming Examples")
    print("=" * 60)
    print("\nThis example demonstrates async streaming logs with asyncio.sleep")
    print("for concurrent real-time log monitoring.\n")

    print("Choose an example:")
    print("1. Single container with async streaming")
    print("2. Multiple containers with concurrent monitoring")
    print("3. Run all examples")

    try:
        choice = input(
            "\nEnter choice (1-3, or just press Enter for example 1): "
        ).strip()

        if choice == "2":
            asyncio.run(async_example_multiple_containers())
        elif choice == "3":
            asyncio.run(async_main())
            print("\n" + "=" * 60 + "\n")
            asyncio.run(async_example_multiple_containers())
        else:
            asyncio.run(async_main())

    except KeyboardInterrupt:
        print("\n\nExamples interrupted by user")
    except Exception as e:
        print(f"\nError: {e}")
