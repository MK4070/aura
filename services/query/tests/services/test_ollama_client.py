import json
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager
from typing import Any
from unittest.mock import AsyncMock, MagicMock

import httpx
import pytest

from app.services import OllamaClient


@pytest.fixture
def ollama_setup() -> tuple[OllamaClient, AsyncMock]:
    mock_http = AsyncMock()
    client = OllamaClient(
        client=mock_http,
        embedding_model="test-embed-model",
        generation_model="test-generation-model",
    )
    return client, mock_http


async def test_is_healthy_success(ollama_setup: tuple[OllamaClient, AsyncMock]) -> None:
    """Test that a 200 OK from the base URL indicates the LLM is up."""
    ollama, mock_http = ollama_setup

    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_http.get.return_value = mock_response

    assert await ollama.is_healthy() is True


async def test_is_healthy_failure(ollama_setup: tuple[OllamaClient, AsyncMock]) -> None:
    """Test that any exception during the ping correctly flags as unhealthy."""
    ollama, mock_http = ollama_setup

    mock_http.get.side_effect = httpx.RequestError(
        "Connection refused", request=MagicMock()
    )
    assert await ollama.is_healthy() is False


async def test_generate_embedding_success(
    ollama_setup: tuple[OllamaClient, AsyncMock],
) -> None:
    """Tests the exact JSON contract Ollama's /api/embed endpoint returns."""
    ollama, mock_http = ollama_setup

    mock_response = MagicMock()
    # The new /api/embed endpoint returns a 2D array of floats
    mock_response.json.return_value = {"embeddings": [[0.1, 0.2, 0.3]]}
    mock_response.raise_for_status.return_value = None
    mock_http.post.return_value = mock_response

    vector = await ollama.embed("How do I deploy?")

    assert vector == [0.1, 0.2, 0.3]
    mock_http.post.assert_called_once()


async def test_generate_embedding_network_error(
    ollama_setup: tuple[OllamaClient, AsyncMock],
) -> None:
    """Verifies that httpx routing errors are caught and re-raised cleanly."""
    ollama, mock_http = ollama_setup

    mock_http.post.side_effect = httpx.RequestError(
        "Cannot route to host", request=MagicMock()
    )

    with pytest.raises(Exception) as exc_info:
        await ollama.embed("Test")

    assert "Failed to communicate with Ollama embedding service" in str(exc_info.value)
    assert "Cannot route to host" in str(exc_info.value)


async def test_stream_generation_success(
    ollama_setup: tuple[OllamaClient, AsyncMock],
) -> None:
    """
    Simulates a raw HTTP streaming connection. We must use an
    asynccontextmanager to replicate `async with httpx.stream(...)`.
    """
    ollama, mock_http = ollama_setup

    @asynccontextmanager
    async def mock_stream(*args: Any, **kwargs: Any) -> AsyncGenerator[AsyncMock, None]:
        mock_response = AsyncMock()
        mock_response.raise_for_status = MagicMock()

        # Simulate Ollama yielding NdJSON
        async def mock_aiter_lines(
            *args: Any, **kwargs: Any
        ) -> AsyncGenerator[str, None]:
            yield json.dumps({"response": "I "})
            yield json.dumps({"response": "am "})
            yield json.dumps({"response": "Aura."})
            yield ""

        mock_response.aiter_lines = mock_aiter_lines
        yield mock_response

    # Overwrite the stream method with our context manager
    mock_http.stream = mock_stream

    streamed_tokens = []
    async for token in ollama.stream_generation("Who are you?"):
        streamed_tokens.append(token)

    assert streamed_tokens == ["I ", "am ", "Aura."]


async def test_stream_generation_http_error(
    ollama_setup: tuple[OllamaClient, AsyncMock],
) -> None:
    """
    Tests graceful error handling if Ollama returns a 4xx/5xx code
    (e.g., Model Not Found) during the stream initialization.
    """
    ollama, mock_http = ollama_setup

    # Request and Response for the HTTPX error
    mock_request = MagicMock()
    mock_response = MagicMock()
    mock_response.status_code = 404
    mock_response.json.return_value = {"error": "model 'llama3' not found"}
    mock_response.aread = AsyncMock()

    http_error = httpx.HTTPStatusError(
        "404 Not Found", request=mock_request, response=mock_response
    )

    # context manager crashing immediately
    @asynccontextmanager
    async def failing_stream(
        *args: Any, **kwargs: Any
    ) -> AsyncGenerator[AsyncMock, None]:
        mock_resp = MagicMock()
        mock_resp.raise_for_status.side_effect = http_error
        yield mock_resp

    mock_http.stream = failing_stream

    with pytest.raises(Exception) as exc_info:
        _ = [token async for token in ollama.stream_generation("Hello")]

    assert "Ollama generation failed" in str(exc_info.value)
    assert "model 'llama3' not found" in str(exc_info.value)
