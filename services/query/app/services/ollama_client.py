import json
from collections.abc import AsyncGenerator
from typing import cast

import httpx
from fastapi import HTTPException, status


class OllamaClient:
    """
    Handles communication with the local Ollama API
    """

    def __init__(
        self, client: httpx.AsyncClient, embedding_model: str, generation_model: str
    ) -> None:
        self.client = client
        self.embedding_model = embedding_model
        self.generation_model = generation_model

    async def embed(self, text: str) -> list[float]:
        """
        Takes a raw string and returns its dense vector representation.
        """
        payload = {"model": self.embedding_model, "input": text}

        try:
            response = await self.client.post("/api/embed", json=payload)
            response.raise_for_status()

            data = response.json()
            return cast(list[float], data["embeddings"][0])

        except httpx.HTTPError as e:
            raise HTTPException(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                detail=f"Failed to communicate with Ollama embedding service: {str(e)}",
            )

    async def stream_generation(self, prompt: str) -> AsyncGenerator[str, None]:
        """
        Sends the formatted prompt to Ollama and yields the response token-by-token
        """
        payload = {"model": self.generation_model, "prompt": prompt, "stream": True}

        try:
            async with self.client.stream(
                "POST",
                "/api/generate",
                json=payload,
                timeout=None,
            ) as response:
                response.raise_for_status()

                async for line in response.aiter_lines():
                    if line:
                        data = json.loads(line)
                        token = data.get("response", "")
                        if token:
                            yield token

        except httpx.HTTPStatusError as e:
            await e.response.aread()
            try:
                error_detail = e.response.json().get("error", str(e))
            except Exception:
                error_detail = str(e)
            raise Exception(f"Ollama generation failed: {error_detail}")

        except httpx.RequestError as e:
            raise Exception(f"Network error communicating with Ollama: {str(e)}")

    async def is_healthy(self) -> bool:
        try:
            response = await self.client.get("/")
            return response.status_code == 200
        except Exception:
            return False

    async def close(self) -> None:
        """Gracefully close the HTTP client pool."""
        await self.client.aclose()
