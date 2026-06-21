import httpx
from fastapi import Depends
from qdrant_client import AsyncQdrantClient

from app.core import Settings, get_settings
from app.repositories import QdrantRepository
from app.services import OllamaClient

_qdrant_client: AsyncQdrantClient | None = None
_httpx_client: httpx.AsyncClient | None = None


# ─── BASE DEPENDENCIES ───
def get_qdrant_client() -> AsyncQdrantClient:
    if _qdrant_client is None:
        raise RuntimeError("Qdrant client is not initialized")
    return _qdrant_client


def get_httpx_client() -> httpx.AsyncClient:
    if _httpx_client is None:
        raise RuntimeError("HTTPX client is not initialized")
    return _httpx_client


# ─── HIGH-LEVEL DEPENDENCIES ───
def get_qdrant_repository(
    settings: Settings = Depends(get_settings),
    client: AsyncQdrantClient = Depends(get_qdrant_client),
) -> QdrantRepository:
    return QdrantRepository(client=client, collection_name=settings.QDRANT_COLLECTION)


def get_ollama_client(
    settings: Settings = Depends(get_settings),
    http_client: httpx.AsyncClient = Depends(get_httpx_client),
) -> OllamaClient:
    return OllamaClient(
        client=http_client,
        embedding_model=settings.EMBEDDING_MODEL,
        generation_model=settings.GENERATION_MODEL,
    )
