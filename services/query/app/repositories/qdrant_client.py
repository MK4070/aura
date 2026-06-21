from fastapi import HTTPException, status
from qdrant_client import AsyncQdrantClient
from qdrant_client.http.exceptions import UnexpectedResponse

from app.models import QdrantPayload, RetrievedChunk


class QdrantRepository:
    """
    Handles querying the qdrant database
    """

    def __init__(self, client: AsyncQdrantClient, collection_name: str) -> None:
        self.client = client
        self.collection_name = collection_name

    async def search_similar_chunks(
        self, query_vector: list[float], top_k: int = 5
    ) -> list[RetrievedChunk]:
        try:
            search_response = await self.client.query_points(
                collection_name=self.collection_name,
                query=query_vector,
                limit=top_k,
                with_payload=True,
                with_vectors=False,
            )

            parsed_chunks: list[RetrievedChunk] = []

            for point in search_response.points:
                payload_data = QdrantPayload(**point.payload)  # type: ignore

                chunk = RetrievedChunk(
                    id=point.id, score=point.score, payload=payload_data
                )
                parsed_chunks.append(chunk)

            return parsed_chunks

        except UnexpectedResponse as e:
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Qdrant query failed: {e.reason_phrase}",
            )
        except ValueError as e:
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Data contract violation between Qdrant and Python: {str(e)}",
            )

    async def is_healthy(self) -> bool:
        try:
            await self.client.get_collections()
            return True
        except Exception:
            return False

    async def close(self) -> None:
        await self.client.close()
