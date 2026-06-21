import json
from collections.abc import AsyncGenerator
from typing import Any
from unittest.mock import AsyncMock
from uuid import uuid4

import pytest
from fastapi import HTTPException

from app.models import QdrantPayload, RetrievedChunk
from app.services import RAGOrchestrator


@pytest.fixture
def orchestrator(mock_ollama: AsyncMock, mock_qdrant: AsyncMock) -> RAGOrchestrator:
    mock_ollama.generation_model = "test-model"

    return RAGOrchestrator(ollama_client=mock_ollama, qdrant_repo=mock_qdrant)


@pytest.fixture
def fake_qdrant_chunk() -> RetrievedChunk:
    """Provides a standardized fake database record for tests."""
    return RetrievedChunk(
        id=uuid4(),
        score=0.95,
        payload=QdrantPayload(
            documentId=uuid4(),
            content="Test context",
            sequenceNumber=1,
            startByte=0,
            endByte=10,
            tokenCount=2,
            fileName="test.txt",
        ),
    )


async def test_generate_stream_success(
    orchestrator: RAGOrchestrator,
    mock_ollama: AsyncMock,
    mock_qdrant: AsyncMock,
    fake_qdrant_chunk: RetrievedChunk,
) -> None:
    """
    GOLDEN PATH: Tests that the orchestrator correctly retrieves context,
    formats the citations, and yields the SSE JSON payloads in the correct order.
    """
    mock_qdrant.search_similar_chunks.return_value = [fake_qdrant_chunk]

    async def mock_token_stream(*args: Any, **kwargs: Any) -> AsyncGenerator[str, None]:
        yield "Test "
        yield "success."

    mock_ollama.stream_generation = mock_token_stream

    stream_generator = await orchestrator.generate_stream(query="Test query", top_k=1)

    events = [event async for event in stream_generator]
    assert all(e.startswith("data: ") and e.endswith("\n\n") for e in events)

    # Extract the JSON strings
    payloads = [json.loads(e.replace("data: ", "").strip()) for e in events]

    # Assert Sequence Contract (Sources -> Tokens -> Done)
    assert len(payloads) == 4

    assert payloads[0]["type"] == "sources"
    assert len(payloads[0]["data"]) == 1
    assert payloads[0]["data"][0]["file_name"] == "test.txt"

    assert payloads[1]["type"] == "token"
    assert payloads[1]["data"] == "Test "

    assert payloads[2]["type"] == "token"
    assert payloads[2]["data"] == "success."

    assert payloads[3]["type"] == "done"


async def test_generate_stream_initialization_failure(
    orchestrator: RAGOrchestrator, mock_ollama: AsyncMock
) -> None:
    """
    EARLY FAILURE: If Qdrant or Ollama fails *before* the stream starts
    (e.g., embedding generation fails), it should raise an HTTP 500.
    """
    mock_ollama.embed.side_effect = Exception("Ollama is down")

    with pytest.raises(HTTPException) as exc_info:
        await orchestrator.generate_stream(query="Test", top_k=1)

    assert exc_info.value.status_code == 500
    assert "Orchestration pipeline failed" in exc_info.value.detail


async def test_generate_stream_mid_stream_crash(
    orchestrator: RAGOrchestrator, mock_ollama: AsyncMock, mock_qdrant: AsyncMock
) -> None:
    """
    MID-STREAM CRASH (Graceful Degradation): If the LLM crashes *while* generating,
    the orchestrator must catch it and yield an {"type": "error"} JSON block
    to the frontend, instead of tearing down the TCP connection ungracefully.
    """

    # Fake a crash after the first token
    async def crashing_token_stream(
        *args: Any, **kwargs: Any
    ) -> AsyncGenerator[str, None]:
        yield "First token..."
        raise Exception("GPU Out of Memory")

    mock_ollama.stream_generation = crashing_token_stream

    stream_generator = await orchestrator.generate_stream(query="Test", top_k=1)
    payloads = [
        json.loads(e.replace("data: ", "").strip()) async for e in stream_generator
    ]

    # Assert Sequence: Sources -> Token -> Error
    assert len(payloads) == 3
    assert payloads[0]["type"] == "sources"
    assert payloads[1]["type"] == "token"

    # Ensure the error was caught and formatted for the UI
    assert payloads[2]["type"] == "error"
    assert "GPU Out of Memory" in payloads[2]["data"]
