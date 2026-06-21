from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field


class QueryRequest(BaseModel):
    """The payload expected from the user/client frontend."""

    query: str = Field(..., min_length=3, description="The user's question.")
    top_k: int = Field(
        default=5,
        ge=1,
        le=20,
        description="Number of relevant chunks to retrieve from Qdrant.",
    )


class QdrantPayload(BaseModel):
    """
    Strict mapping of the flattened payload created by the ingestion worker
    """

    documentId: UUID
    content: str
    sequenceNumber: int
    startByte: int
    endByte: int
    tokenCount: int

    # Metadata fields
    fileName: str | None = None
    uploadedAt: str | None = None
    sourceType: str | None = None

    # We allow extra fields just in case worker adds new metadata
    # in the future that we haven't strictly typed here yet
    model_config = ConfigDict(extra="allow")


class RetrievedChunk(BaseModel):
    """
    The domain model representing a scored result from Qdrant.
    """

    id: UUID | int | str
    score: float
    payload: QdrantPayload


# synchronous fallback or debug endpoint
class QueryResponse(BaseModel):
    answer: str
    sources: list[RetrievedChunk]
