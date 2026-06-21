from collections.abc import AsyncGenerator, Generator
from unittest.mock import AsyncMock

import pytest
from httpx import ASGITransport, AsyncClient
from qdrant_client import AsyncQdrantClient
from qdrant_client.models import Distance, PointStruct, VectorParams
from testcontainers.qdrant import QdrantContainer

from app import app
from app.core import get_settings
from app.dependencies import get_ollama_client, get_qdrant_repository
from app.repositories import QdrantRepository

settings = get_settings()


@pytest.fixture(scope="module")
def qdrant_container() -> Generator[QdrantContainer, None, None]:
    """
    Spins up a qdrant container. scope is set to 'module' so it stays
    alive for all tests in this file.
    """
    with QdrantContainer(image="qdrant/qdrant:v1.18") as qdrant:
        yield qdrant


@pytest.fixture
def override_settings(qdrant_container: QdrantContainer) -> None:
    """
    Injects the ephemeral Docker port into our application settings
    """
    host = qdrant_container.get_container_host_ip()
    port = qdrant_container.get_exposed_port(6333)

    settings.QDRANT_HOST = host
    settings.QDRANT_PORT = int(port)


@pytest.fixture
async def seeded_qdrant_repo(
    override_settings: None,
) -> AsyncGenerator[QdrantRepository, None]:
    """
    Initializes a real QdrantRepository, creates a collection,
    and seeds it with data.
    """
    collection_name = settings.QDRANT_COLLECTION

    # 1. Create a REAL client pointing to the testcontainer
    client = AsyncQdrantClient(
        url=f"http://{settings.QDRANT_HOST}:{settings.QDRANT_PORT}",
        check_compatibility=False,
    )

    # 2. Instantiate a REAL repository
    repo = QdrantRepository(client=client, collection_name=collection_name)

    await repo.client.create_collection(
        collection_name=collection_name,
        vectors_config=VectorParams(size=3, distance=Distance.COSINE),
    )

    point = PointStruct(
        id="123e4567-e89b-12d3-a456-426614174000",
        vector=[0.1, 0.2, 0.3],
        payload={
            "documentId": "987e6543-e21b-34d3-b456-426614174999",
            "content": "This is real data inside a Docker container.",
            "sequenceNumber": 1,
            "startByte": 0,
            "endByte": 42,
            "tokenCount": 8,
            "fileName": "integration_guide.md",
        },
    )
    await repo.client.upsert(collection_name=collection_name, points=[point])

    yield repo

    # Teardown
    await repo.client.delete_collection(collection_name=collection_name)
    await repo.client.close()  # Make sure we close the inner network client


async def test_qdrant_retrieval_flow(
    seeded_qdrant_repo: QdrantRepository, mock_ollama: AsyncMock
) -> None:
    """
    E2E infra test.
    """
    # Override FastAPI dependencies for this specific test
    app.dependency_overrides[get_qdrant_repository] = lambda: seeded_qdrant_repo
    app.dependency_overrides[get_ollama_client] = lambda: mock_ollama

    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://testserver") as client:
        payload = {"query": "Fetch real data", "top_k": 1}
        response = await client.post("/api/v1/query", json=payload)

        assert response.status_code == 200
        content = response.text

        assert "integration_guide.md" in content
        assert '"type": "sources"' in content

    # Clean up overrides so they don't leak into other tests
    app.dependency_overrides.clear()
