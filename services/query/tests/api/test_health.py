from unittest.mock import AsyncMock

from httpx import AsyncClient


async def test_health_check_healthy(async_client: AsyncClient) -> None:
    """Test the health endpoint when all dependencies are up."""
    response = await async_client.get("/health")

    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "healthy"
    assert data["components"]["ollama"] == "up"
    assert data["components"]["qdrant"] == "up"


async def test_health_check_unhealthy_qdrant(
    async_client: AsyncClient, mock_qdrant: AsyncMock
) -> None:
    """Test the health endpoint when Qdrant goes down."""
    mock_qdrant.is_healthy.return_value = False

    response = await async_client.get("/health")

    assert response.status_code == 503
    data = response.json()
    assert data["status"] == "unhealthy"
    assert data["components"]["qdrant"] == "down"
    assert data["components"]["ollama"] == "up"


async def test_health_check_unhealthy_ollama(
    async_client: AsyncClient, mock_ollama: AsyncMock
) -> None:
    """Test the health endpoint when Ollama goes down."""
    mock_ollama.is_healthy.return_value = False

    response = await async_client.get("/health")

    assert response.status_code == 503
    data = response.json()
    assert data["status"] == "unhealthy"
    assert data["components"]["qdrant"] == "up"
    assert data["components"]["ollama"] == "down"


async def test_health_check_all_unhealthy(
    async_client: AsyncClient, mock_qdrant: AsyncMock, mock_ollama: AsyncMock
) -> None:
    """Test the health endpoint when both services crash."""
    mock_qdrant.is_healthy.return_value = False
    mock_ollama.is_healthy.return_value = False

    response = await async_client.get("/health")

    assert response.status_code == 503
    data = response.json()
    assert data["status"] == "unhealthy"
    assert data["components"]["qdrant"] == "down"
    assert data["components"]["ollama"] == "down"
