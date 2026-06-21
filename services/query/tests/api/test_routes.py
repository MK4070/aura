from unittest.mock import AsyncMock

from httpx import AsyncClient


async def test_query_streaming_success(async_client: AsyncClient) -> None:
    """
    Tests the golden path: A valid query results in a properly formatted
    Server-Sent Events (SSE) stream containing sources, tokens, and a done signal.
    """
    payload = {"query": "How do I deploy to Kubernetes?", "top_k": 2}

    response = await async_client.post("/api/v1/query", json=payload)

    assert response.status_code == 200
    assert "text/event-stream" in response.headers["content-type"]

    content = response.text

    # Citation Payload
    assert '"type": "sources"' in content
    assert '"file_name": "k8s_guide.txt"' in content
    assert '"score": 0.98' in content

    # Token Payload (FIX: Updated to match the mock_ollama stream)
    assert '"type": "token", "data": "This is "' in content
    assert '"type": "token", "data": "a mocked "' in content
    assert '"type": "token", "data": "LLM response."' in content

    # Termination Payload
    assert '"type": "done"' in content


async def test_query_validation_errors(async_client: AsyncClient) -> None:
    """
    Ensures our Pydantic validation correctly rejects malformed requests
    before they ever hit the database or LLM.
    """
    # Missing required 'query' field
    bad_payload_1 = {"top_k": 3}
    response_1 = await async_client.post("/api/v1/query", json=bad_payload_1)
    assert response_1.status_code == 422

    # Query string too short (violates min_length=3)
    bad_payload_2 = {"query": "Hi", "top_k": 3}
    response_2 = await async_client.post("/api/v1/query", json=bad_payload_2)
    assert response_2.status_code == 422


async def test_query_orchestrator_failure(
    async_client: AsyncClient, mock_ollama: AsyncMock
) -> None:
    """
    Tests that if a dependency crashes BEFORE the stream starts,
    the API returns a clean 500 error instead of hanging.
    """
    # Force the embedding generation to raise an exception
    mock_ollama.embed.side_effect = Exception("Ollama is offline")

    payload = {"query": "Will this crash?", "top_k": 1}
    response = await async_client.post("/api/v1/query", json=payload)

    assert response.status_code == 500
    assert "Orchestration pipeline failed" in response.json()["detail"]
