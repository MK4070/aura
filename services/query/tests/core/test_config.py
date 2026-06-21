import os

from pytest import MonkeyPatch

from app.core.config import Settings


def test_settings_defaults() -> None:
    """
    Test that default settings are applied when environment variables are missing.
    """
    # Temporarily remove an env var if it exists in the test runner
    original_host = os.environ.get("QDRANT_HOST")
    if "QDRANT_HOST" in os.environ:
        del os.environ["QDRANT_HOST"]

    settings = Settings()  # type: ignore

    # Verify fallback behavior (assuming your default is localhost)
    assert settings.QDRANT_HOST == "localhost"
    assert settings.API_V1_STR == "/api/v1"

    # Restore the environment variable
    if original_host:
        os.environ["QDRANT_HOST"] = original_host


def test_settings_override(monkeypatch: MonkeyPatch) -> None:
    """
    Test that explicit environment variables override the defaults.
    """
    monkeypatch.setenv("PROJECT_NAME", "Aura_Test_Env")
    monkeypatch.setenv("QDRANT_PORT", "9999")

    settings = Settings()  # type: ignore

    assert settings.PROJECT_NAME == "Aura_Test_Env"
    assert settings.QDRANT_PORT == 9999  # Ensures type coercion from string to int
