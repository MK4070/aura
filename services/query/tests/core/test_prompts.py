from app.core import RAG_SYSTEM_PROMPT


def test_rag_system_prompt_format() -> None:
    """
    Ensures the system prompt contains the required placeholder for string formatting.
    If this fails, the orchestrator will crash at runtime.
    """
    assert "{context}" in RAG_SYSTEM_PROMPT, (
        "Prompt is missing the {context} placeholder"
    )

    # Verify it can be formatted without throwing a KeyError
    test_context = "This is a retrieved document."
    formatted_prompt = RAG_SYSTEM_PROMPT.format(context=test_context)

    assert test_context in formatted_prompt
    assert (
        "strictly" in formatted_prompt or "STRICTLY" in formatted_prompt
    )  # Validates rules are present
