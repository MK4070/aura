from typing import Any

from fastapi import APIRouter, Depends, Response, status
from pydantic import BaseModel

from app.dependencies import get_ollama_client, get_qdrant_repository
from app.repositories import QdrantRepository
from app.services import OllamaClient

router = APIRouter()


class HealthResponse(BaseModel):
    status: str
    components: dict[str, str]


@router.get(
    "/health",
    response_model=HealthResponse,
    summary="Deep health check for readiness probes",
)
async def health_check(
    response: Response,
    ollama_client: OllamaClient = Depends(get_ollama_client),
    qdrant_repo: QdrantRepository = Depends(get_qdrant_repository),
) -> dict[str, Any]:
    """
    Checks the status of the Python server and its critical dependencies.
    Returns a 503 if any downstream component is unreachable.
    """
    ollama_ok = await ollama_client.is_healthy()
    qdrant_ok = await qdrant_repo.is_healthy()

    components = {
        "ollama": "up" if ollama_ok else "down",
        "qdrant": "up" if qdrant_ok else "down",
    }

    if not all([ollama_ok, qdrant_ok]):
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE
        return {"status": "unhealthy", "components": components}

    return {"status": "healthy", "components": components}
