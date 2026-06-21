import json
from collections.abc import AsyncGenerator

from fastapi import HTTPException, status
from opentelemetry import trace

from app.core import LLM_TOKENS_GENERATED, RAG_SYSTEM_PROMPT
from app.repositories import QdrantRepository
from app.services import OllamaClient

tracer = trace.get_tracer(__name__)


class RAGOrchestrator:
    """
    Coordinates the Retrieval-Augmented Generation pipeline
    """

    def __init__(self, ollama_client: OllamaClient, qdrant_repo: QdrantRepository):
        self.ollama = ollama_client
        self.qdrant = qdrant_repo

    async def generate_stream(
        self, query: str, top_k: int
    ) -> AsyncGenerator[str, None]:
        """
        Executes the RAG pipeline and returns an async token generator
        """
        try:
            with tracer.start_as_current_span("embed") as span:
                span.set_attribute("query.length", len(query))
                query_vector = await self.ollama.embed(query)

            with tracer.start_as_current_span("qdrant_retrieve") as span:
                span.set_attribute("top_k", top_k)
                retrieved_chunks = await self.qdrant.search_similar_chunks(
                    query_vector=query_vector, top_k=top_k
                )
                span.set_attribute("chunks_found", len(retrieved_chunks))

            context_blocks = []
            citations = []

            for chunk in retrieved_chunks:
                source_name = chunk.payload.fileName or "Unknown Source"
                content = chunk.payload.content

                context_blocks.append(f"--- Source: {source_name} ---\n{content}")

                citations.append(
                    {
                        "id": str(chunk.id),
                        "file_name": source_name,
                        "score": chunk.score,
                    }
                )

            joined_context = "\n\n".join(context_blocks)
            final_prompt = RAG_SYSTEM_PROMPT.format(context=joined_context)
            final_prompt += f"\n\nUser Question: {query}\nAnswer:"

            async def sse_generator() -> AsyncGenerator[str, None]:
                try:
                    sources_payload = json.dumps({"type": "sources", "data": citations})
                    yield f"data: {sources_payload}\n\n"

                    token_stream = self.ollama.stream_generation(prompt=final_prompt)
                    async for token in token_stream:
                        token_payload = json.dumps({"type": "token", "data": token})
                        yield f"data: {token_payload}\n\n"

                        LLM_TOKENS_GENERATED.add(
                            1, {"model": self.ollama.generation_model}
                        )

                    done_payload = json.dumps({"type": "done"})
                    yield f"data: {done_payload}\n\n"

                except Exception as e:
                    error_payload = json.dumps({"type": "error", "data": str(e)})
                    yield f"data: {error_payload}\n\n"

            return sse_generator()

        except Exception as e:
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Orchestration pipeline failed: {str(e)}",
            )
