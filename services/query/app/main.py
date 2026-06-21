import os
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager

import httpx
from fastapi import FastAPI
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles
from qdrant_client import AsyncQdrantClient

import app.dependencies as deps
from app.api import api_router, health_router
from app.core import get_settings, setup_telemetry


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator[None, None]:
    # Startup
    settings = get_settings()

    deps._qdrant_client = AsyncQdrantClient(
        url=f"http://{settings.QDRANT_HOST}:{settings.QDRANT_PORT}",
        check_compatibility=False,
    )
    deps._httpx_client = httpx.AsyncClient(
        base_url=settings.OLLAMA_BASE_URL, timeout=30.0
    )

    yield

    # Shutdown
    if deps._httpx_client:
        await deps._httpx_client.aclose()
    if deps._qdrant_client:
        await deps._qdrant_client.close()


settings = get_settings()

app = FastAPI(
    title=settings.PROJECT_NAME,
    lifespan=lifespan,
    docs_url=f"{settings.API_V1_STR}/docs",
)

setup_telemetry(app)

# app.add_middleware(
#     CORSMiddleware,
#     allow_origins=["*"],
#     allow_credentials=True,
#     allow_methods=["*"],
#     allow_headers=["*"],
# )

app.include_router(api_router, prefix=settings.API_V1_STR)
app.include_router(health_router, tags=["Diagnostics"])

# === UI ===
os.makedirs("app/static", exist_ok=True)
app.mount("/static", StaticFiles(directory="app/static"), name="static")


@app.get("/", summary="Serve the Native Chat UI")
async def serve_ui() -> FileResponse:
    """Serves the frontend"""
    return FileResponse("app/static/index.html")
