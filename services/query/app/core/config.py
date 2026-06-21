from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    # Server Configuration
    PROJECT_NAME: str = "Aura Query Service"
    API_V1_STR: str = "/api/v1"
    ENVIRONMENT: str = "development"

    # Qdrant
    QDRANT_HOST: str = "localhost"
    QDRANT_PORT: int = 6333
    QDRANT_COLLECTION: str = "document"

    # Ollama
    OLLAMA_BASE_URL: str = "http://localhost:11434"
    EMBEDDING_MODEL: str = "nomic-embed-text"
    GENERATION_MODEL: str = "llama3.2:3b"

    # OTLP
    OTLP_ENDPOINT: str = "http://localhost:4317"

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=True,
        extra="ignore",
    )


@lru_cache
def get_settings() -> Settings:
    """
    Returns a cached instance of the Settings object.
    Uses lru_cache to ensure the .env file is only read once.
    """
    return Settings()
