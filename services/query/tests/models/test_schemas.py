import pytest
from pydantic import ValidationError

from app.models import QueryRequest


def test_query_request_valid() -> None:
    """Ensure valid data passes validation."""
    req = QueryRequest(query="How do I deploy?", top_k=5)
    assert req.query == "How do I deploy?"
    assert req.top_k == 5


def test_query_request_invalid_top_k() -> None:
    """Ensure top_k cannot exceed our defined maximum (20)."""
    with pytest.raises(ValidationError) as exc_info:
        QueryRequest(query="How do I deploy?", top_k=50)

    assert "Input should be less than or equal to 20" in str(exc_info.value)


def test_query_request_too_short() -> None:
    """Ensure queries must be at least 3 characters."""
    with pytest.raises(ValidationError):
        QueryRequest(query="Hi", top_k=5)
