"""Streaming example with time.sleep for periodic log retrieval."""

import time
import json
from app.client import SandboxClient


def stream_logs_with_polling(container_id: str, interval: float = 0.5):
    """Stream logs using time.sleep for periodic polling.

    Args:
        container_id: Container ID to stream logs from
        interval: Sleep interval between log fetches (seconds)
    """
    client = SandboxClient("http://localhost:8000")

    try:
        print(f"Starting log streaming for container {container_id}...")
        print("Press Ctrl+C to stop\n")

        start_time = time.time()
        timeout = 30  # Maximum streaming duration

        while time.time() - start_time < timeout:
            try:
                # Open streaming connection
                with client.stream_logs(container_id) as response:
                    if response.status_code == 200:
                        # Read available data
                        for line in response.iter_lines():
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
                # No more data for now, wait before next poll
                time.sleep(interval)

            except KeyboardInterrupt:
                print("\n\nStopping log streaming...")
                break
            except Exception as e:
                print(f"Error streaming logs: {e}")
                time.sleep(interval)

    finally:
        client.close()


def main():
    """Main example demonstrating container execution with log streaming."""
    client = None
    container_id = None

    try:
        client = SandboxClient("http://localhost:8000")

        # Example 1: Simple output with delays
        print("=== Example 1: Simple delayed output ===\n")

        code = """import time

print("Starting execution...")
for i in range(5):
    print(f"Processing step {i+1}/5")
    time.sleep(1)
print("Execution complete!")
"""

        # Create container
        container_id = client.create_container(image="python:3.12-alpine", code=code)
        print(f"Created container: {container_id}\n")

        # Start container
        client.start_container(container_id=container_id, code=code, timeout=30)
        print("Container started, beginning log streaming...\n")

        # Stream logs with polling
        time.sleep(0.5)  # Initial delay to let container start
        stream_logs_with_polling(container_id, interval=0.3)

        # Cleanup
        print("\nStopping container...")
        client.stop_container(container_id)
        print("Done!")

    except Exception as e:
        print(f"Error: {e}")
        if container_id and client:
            try:
                client.stop_container(container_id)
            except Exception:
                pass
    finally:
        if client:
            client.close()


def example2_progressive_output():
    """Example 2: Progressive output with progress indication."""
    client = None
    container_id = None

    try:
        client = SandboxClient("http://localhost:8000")

        print("\n=== Example 2: Progressive output ===\n")

        code = """import time
import sys

tasks = ["Initialize", "Load data", "Process data", "Validate results", "Save output"]
for i, task in enumerate(tasks):
    print(f"[{i+1}/{len(tasks)}] {task}...", flush=True)
    time.sleep(0.8)
    print(f"[{i+1}/{len(tasks)}] {task} - DONE", flush=True)
print("All tasks completed!")
"""

        container_id = client.create_container(image="python:3.12-alpine", code=code)
        print(f"Created container: {container_id}\n")

        client.start_container(container_id=container_id, code=code, timeout=30)
        print("Container started\n")

        # Stream logs
        time.sleep(0.5)
        start_time = time.time()

        while time.time() - start_time < 15:
            try:
                with client.stream_logs(container_id) as response:
                    if response.status_code == 200:
                        for line in response.iter_lines():
                            if line.startswith("data: "):
                                json_str = line[6:]
                                try:
                                    msg = json.loads(json_str)
                                    if "data" in msg and msg["data"]:
                                        print(msg["data"].strip())
                                except json.JSONDecodeError:
                                    continue
                time.sleep(0.2)

            except KeyboardInterrupt:
                break
            except Exception as e:
                print(f"Error: {e}")
                time.sleep(0.2)

        print("\nStopping container...")
        client.stop_container(container_id)

    except Exception as e:
        print(f"Error: {e}")
        if container_id and client:
            try:
                client.stop_container(container_id)
            except Exception:
                pass
    finally:
        if client:
            client.close()


def example3_realtime_monitoring():
    """Example 3: Real-time monitoring with timestamps."""
    client = None
    container_id = None

    try:
        client = SandboxClient("http://localhost:8000")

        print("\n=== Example 3: Real-time monitoring ===\n")

        code = """import time

print("Monitoring system status...")
for i in range(10):
    print(f"[{time.strftime('%H:%M:%S')}] Status check {i+1}: All systems operational")
    time.sleep(1.5)
print("Monitoring session ended")
"""

        container_id = client.create_container(image="python:3.12-alpine", code=code)
        print(f"Created container: {container_id}\n")

        client.start_container(container_id=container_id, code=code, timeout=30)
        print("Container started\n")
        print("-" * 60)

        # Stream logs with formatted timestamps
        time.sleep(0.5)
        session_start = time.time()

        while time.time() - session_start < 20:
            try:
                with client.stream_logs(container_id) as response:
                    if response.status_code == 200:
                        for line in response.iter_lines():
                            if line.startswith("data: "):
                                json_str = line[6:]
                                try:
                                    msg = json.loads(json_str)
                                    if "data" in msg and msg["data"]:
                                        elapsed = time.time() - session_start
                                        print(
                                            f"[T+{elapsed:5.1f}s] {msg['data'].strip()}"
                                        )
                                except json.JSONDecodeError:
                                    continue
                time.sleep(0.3)

            except KeyboardInterrupt:
                print("\n\nMonitoring stopped by user")
                break
            except Exception as e:
                print(f"Error: {e}")
                time.sleep(0.3)

        print("-" * 60)
        print("\nStopping container...")
        client.stop_container(container_id)

    except Exception as e:
        print(f"Error: {e}")
        if container_id and client:
            try:
                client.stop_container(container_id)
            except Exception:
                pass
    finally:
        if client:
            client.close()


if __name__ == "__main__":
    print("Docker Sandbox - Streaming Examples")
    print("=" * 60)
    print("\nThis example demonstrates streaming logs with time.sleep")
    print("for practical real-time log monitoring.\n")

    print("Choose an example:")
    print("1. Simple delayed output")
    print("2. Progressive output with progress indication")
    print("3. Real-time monitoring with timestamps")
    print("4. Run all examples")

    try:
        choice = input(
            "\nEnter choice (1-4, or just press Enter for example 1): "
        ).strip()

        if choice == "2":
            example2_progressive_output()
        elif choice == "3":
            example3_realtime_monitoring()
        elif choice == "4":
            main()
            example2_progressive_output()
            example3_realtime_monitoring()
        else:
            main()

    except KeyboardInterrupt:
        print("\n\nExamples interrupted by user")
    except Exception as e:
        print(f"\nError: {e}")
