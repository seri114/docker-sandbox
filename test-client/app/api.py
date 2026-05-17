"""FastAPI application for sandbox test client."""

import os
import uuid

import httpx
from fastapi import Depends, FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel

from app.client import SandboxClient

# Environment variables
SANDBOX_CONTROLLER_URL = os.getenv(
    "SANDBOX_CONTROLLER_URL", "http://sandbox-controller:8080"
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

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# In-memory storage for execution state (simplified)
executions: dict[str, str] = {}


# Dependency
def get_sandbox_client() -> SandboxClient:
    """Get SandboxClient instance."""
    return SandboxClient(base_url=SANDBOX_CONTROLLER_URL)


# Routes
@app.post("/api/execute", response_model=ExecuteResponse)
async def create_execution(
    request: ExecuteRequest, client: SandboxClient = Depends(get_sandbox_client)
) -> ExecuteResponse:
    """Create a new code execution.

    Creates a container, starts execution, and returns an execution_id.
    """
    # Create container with code
    container_id = client.create_container(image=request.image, code=request.code)

    # Start execution
    client.start_container(
        container_id=container_id, code=request.code, timeout=request.timeout
    )

    # Generate execution ID and store container mapping
    execution_id = str(uuid.uuid4())
    executions[execution_id] = container_id

    return ExecuteResponse(execution_id=execution_id)


@app.get("/api/execute/stream")
async def stream_execution_logs(execution_id: str):
    """Stream execution logs via Server-Sent Events.

    After streaming completes, the container is cleaned up.
    """
    if execution_id not in executions:
        raise HTTPException(status_code=404, detail="Execution not found")

    container_id = executions[execution_id]

    async def event_generator():
        """Generate SSE events from container logs."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            try:
                # Stream logs from container
                async with client.stream(
                    "GET",
                    f"{SANDBOX_CONTROLLER_URL}/containers/logs",
                    params={"id": container_id},
                ) as response:
                    # Read raw chunks and forward immediately
                    async for line in response.aiter_lines():
                        if line:
                            # line is already decoded str
                            # Forward the SSE line as-is with proper SSE termination
                            yield line + "\n\n"
            finally:
                # Remove execution record
                executions.pop(execution_id, None)
                yield "data: [DONE]\n\n"

    return StreamingResponse(
        event_generator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
        },
    )


class CancelRequest(BaseModel):
    """Request model for cancelling execution."""

    execution_id: str


@app.post("/api/execute/cancel")
async def cancel_execution(request: CancelRequest):
    """Cancel a running execution by stopping the container.

    Stops the container and removes the execution record.
    """
    execution_id = request.execution_id

    if execution_id not in executions:
        raise HTTPException(status_code=404, detail="Execution not found")

    container_id = executions.pop(execution_id)

    # Stop the container
    async with httpx.AsyncClient(timeout=30.0) as client:
        try:
            await client.post(
                f"{SANDBOX_CONTROLLER_URL}/containers/stop",
                json={"container_id": container_id},
            )
        except httpx.HTTPError:
            # Container may have already stopped
            pass

    return {"status": "cancelled", "execution_id": execution_id}


# Mount static files
app.mount("/static", StaticFiles(directory="static"), name="static")
