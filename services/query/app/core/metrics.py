from opentelemetry import metrics

meter = metrics.get_meter(__name__)

LLM_TOKENS_GENERATED = meter.create_counter(
    "llm_tokens_generated_total",
    description="Total number of tokens streamed to the user",
)
