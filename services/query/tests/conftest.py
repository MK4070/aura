from collections.abc import AsyncGenerator
from unittest.mock import AsyncMock
from uuid import uuid4

import pytest
from httpx import ASGITransport, AsyncClient

from app import app
from app.dependencies import get_ollama_client, get_qdrant_repository
from app.models import QdrantPayload, RetrievedChunk
from app.repositories import QdrantRepository
from app.services import OllamaClient


@pytest.fixture
def mock_ollama() -> AsyncMock:
    mock = AsyncMock(spec=OllamaClient)
    mock.is_healthy.return_value = True
    mock.embed.return_value = [0.1, 0.2, 0.3]
    mock.generation_model = "test-model"

    async def fake_stream(*args, **kwargs) -> AsyncGenerator[str, None]:
        yield "This is "
        yield "a mocked "
        yield "LLM response."

    mock.stream_generation = fake_stream
    return mock


@pytest.fixture
def mock_qdrant() -> AsyncMock:
    mock = AsyncMock(spec=QdrantRepository)
    mock.is_healthy.return_value = True

    fake_chunk = RetrievedChunk(
        id=uuid4(),
        score=0.98,
        payload=QdrantPayload(
            documentId=uuid4(),
            content="Kubernetes deployments require a YAML configuration.",
            sequenceNumber=1,
            startByte=0,
            endByte=50,
            tokenCount=8,
            fileName="k8s_guide.txt",
        ),
    )
    mock.search_similar_chunks.return_value = [fake_chunk]
    return mock


@pytest.fixture
async def async_client(
    mock_ollama: AsyncMock, mock_qdrant: AsyncMock
) -> AsyncGenerator[AsyncClient, None]:
    app.dependency_overrides[get_ollama_client] = lambda: mock_ollama
    app.dependency_overrides[get_qdrant_repository] = lambda: mock_qdrant

    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://testserver") as client:
        yield client

    # Clean up overrides after the test finishes
    app.dependency_overrides.clear()
