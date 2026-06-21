from fastapi import APIRouter, Depends
from fastapi.responses import StreamingResponse

from app.dependencies import get_ollama_client, get_qdrant_repository
from app.models import QueryRequest
from app.repositories import QdrantRepository
from app.services import OllamaClient, RAGOrchestrator

router = APIRouter()


def get_orchestrator(
    ollama_client: OllamaClient = Depends(get_ollama_client),
    qdrant_repo: QdrantRepository = Depends(get_qdrant_repository),
) -> RAGOrchestrator:
    """Dependency injector for the orchestrator."""
    return RAGOrchestrator(
        ollama_client=ollama_client,
        qdrant_repo=qdrant_repo,
    )


@router.post("/query", summary="Stream a RAG response based on a user query")
async def query_rag(
    request: QueryRequest, orchestrator: RAGOrchestrator = Depends(get_orchestrator)
) -> StreamingResponse:
    """
    Takes a user query, retrieves relevant document chunks from Qdrant,
    and streams the LLM generation back to the client via SSE.
    """
    token_generator = await orchestrator.generate_stream(
        query=request.query, top_k=request.top_k
    )

    return StreamingResponse(token_generator, media_type="text/event-stream")
