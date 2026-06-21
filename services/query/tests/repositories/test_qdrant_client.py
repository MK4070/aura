from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest
from fastapi import HTTPException
from httpx import Headers
from qdrant_client.http.exceptions import UnexpectedResponse

from app.repositories import QdrantRepository


@pytest.fixture
def mock_qdrant_client() -> AsyncMock:
    return AsyncMock()


@pytest.fixture
def mock_qdrant_repo(mock_qdrant_client: AsyncMock) -> QdrantRepository:
    """
    Instantiates the repository by directly injecting the mock client
    and a fake collection name.
    """
    return QdrantRepository(
        client=mock_qdrant_client, collection_name="test_collection"
    )


async def test_is_healthy_success(
    mock_qdrant_repo: QdrantRepository, mock_qdrant_client: AsyncMock
) -> None:
    """Test that a successful collections fetch returns True."""
    mock_qdrant_client.get_collections.return_value = MagicMock()
    assert await mock_qdrant_repo.is_healthy() is True


async def test_is_healthy_failure(
    mock_qdrant_repo: QdrantRepository, mock_qdrant_client: AsyncMock
) -> None:
    """Test that network exceptions during health checks return False."""
    mock_qdrant_client.get_collections.side_effect = Exception("Network timeout")
    assert await mock_qdrant_repo.is_healthy() is False


async def test_search_similar_chunks_success(
    mock_qdrant_repo: QdrantRepository, mock_qdrant_client: AsyncMock
) -> None:
    # 1. Construct the fake Qdrant response object
    fake_point = MagicMock()
    fake_point.id = str(uuid4())
    fake_point.score = 0.99
    fake_point.payload = {
        "documentId": str(uuid4()),
        "content": "This is a test chunk of text.",
        "sequenceNumber": 5,
        "startByte": 100,
        "endByte": 150,
        "tokenCount": 10,
        "fileName": "architecture.md",
    }

    fake_response = MagicMock()
    fake_response.points = [fake_point]

    mock_qdrant_client.query_points.return_value = fake_response

    # 2. Execute the search
    dummy_vector = [0.1] * 768
    chunks = await mock_qdrant_repo.search_similar_chunks(
        query_vector=dummy_vector, top_k=1
    )

    # 3. Assert the data contract was respected
    assert len(chunks) == 1
    assert chunks[0].payload.fileName == "architecture.md"


async def test_search_similar_chunks_qdrant_error(
    mock_qdrant_repo: QdrantRepository, mock_qdrant_client: AsyncMock
) -> None:
    mock_qdrant_client.query_points.side_effect = UnexpectedResponse(
        status_code=400,
        reason_phrase="Vector dimension mismatch",
        content=b"",
        headers=Headers(),
    )

    dummy_vector = [0.1] * 768
    with pytest.raises(HTTPException) as exc_info:
        await mock_qdrant_repo.search_similar_chunks(query_vector=dummy_vector)

    assert exc_info.value.status_code == 500


async def test_search_similar_chunks_validation_error(
    mock_qdrant_repo: QdrantRepository, mock_qdrant_client: AsyncMock
) -> None:
    fake_point = MagicMock()
    fake_point.id = str(uuid4())
    fake_point.score = 0.85
    fake_point.payload = {
        "documentId": str(uuid4()),
        "sequenceNumber": "not_an_int",
    }

    fake_response = MagicMock()
    fake_response.points = [fake_point]

    mock_qdrant_client.query_points.return_value = fake_response

    dummy_vector = [0.1] * 768
    with pytest.raises(HTTPException) as exc_info:
        await mock_qdrant_repo.search_similar_chunks(query_vector=dummy_vector)

    assert exc_info.value.status_code == 500
