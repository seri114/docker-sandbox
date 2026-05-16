"""FastAPI application for sandbox test client."""

import os
import uuid
from typing import Generator

from fastapi import FastAPI, HTTPException, Depends
from fastapi.responses import StreamingResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel

from app.client import SandboxClient


# Environment variables
SANDBOX_CONTROLLER_URL = os.getenv(
    "SANDBOX_CONTROLLER_URL",
    "http://sandbox-controller:8080"
)


# Pydantic models
class ExecuteRequest(BaseModel):
    """Request model for code execution."""
    code: str
    image: str = "python:3.12-alpine"
    timeout: int = 30


class ExecuteResponse(BaseModel):
    """Response model for code execution initiation."""
    execution_id: str


# Initialize FastAPI app
app = FastAPI(title="Sandbox Test Client")

# In-memory storage for execution state (simplified)
executions: dict[str, str] = {}


# Dependency
def get_sandbox_client() -> SandboxClient:
    """Get SandboxClient instance."""
    return SandboxClient(base_url=SANDBOX_CONTROLLER_URL)


# Routes
@app.post("/api/execute", response_model=ExecuteResponse)
async def create_execution(
    request: ExecuteRequest,
    client: SandboxClient = Depends(get_sandbox_client)
) -> ExecuteResponse:
    """Create a new code execution.

    Creates a container, starts execution, and returns an execution_id.
    """
    # Create container
    container_id = client.create_container(image=request.image)

    # Start execution
    client.start_container(
        container_id=container_id,
        code=request.code,
        timeout=request.timeout
    )

    # Generate execution ID and store container mapping
    execution_id = str(uuid.uuid4())
    executions[execution_id] = container_id

    return ExecuteResponse(execution_id=execution_id)


@app.get("/api/execute/stream")
async def stream_execution_logs(
    execution_id: str,
    client: SandboxClient = Depends(get_sandbox_client)
):
    """Stream execution logs via Server-Sent Events.

    After streaming completes, the container is cleaned up.
    """
    if execution_id not in executions:
        raise HTTPException(status_code=404, detail="Execution not found")

    container_id = executions[execution_id]

    async def event_generator():
        """Generate SSE events from container logs."""
        try:
            # Stream logs from container
            with client.stream_logs(container_id) as response:
                for line in response.iter_lines():
                    if line:
                        # SSE format: "data: <message>\n\n"
                        decoded = line.decode("utf-8", errors="replace")
                        yield f"data: {decoded}\n\n"
        finally:
            # Cleanup: remove execution record
            executions.pop(execution_id, None)

    return StreamingResponse(
        event_generator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
        }
    )


# Mount static files
app.mount("/static", StaticFiles(directory="static"), name="static")
