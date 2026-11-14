from __future__ import annotations

from ..config import Settings
from .embeddings import EmbeddingProvider, FakeEmbedder, GeminiEmbedder, TitanEmbedder
from .llm import BedrockProvider, FakeProvider, GeminiProvider, LLMProvider, ObservedProvider


def build_llm_provider(settings: Settings) -> LLMProvider:
    if settings.llm_provider == "gemini":
        inner = GeminiProvider(settings)
    elif settings.llm_provider == "bedrock":
        inner = BedrockProvider(settings)
    elif settings.llm_provider == "fake":
        inner = FakeProvider()
    else:
        raise ValueError(f"unsupported AI_LLM_PROVIDER: {settings.llm_provider}")
    return ObservedProvider(settings.llm_provider, inner)


def build_embedding_provider(settings: Settings) -> EmbeddingProvider:
    if settings.embedding_provider == "gemini":
        return GeminiEmbedder(settings)
    if settings.embedding_provider == "titan":
        return TitanEmbedder(settings)
    if settings.embedding_provider == "fake":
        return FakeEmbedder(settings.embedding_dim)
    raise ValueError(f"unsupported AI_EMBEDDING_PROVIDER: {settings.embedding_provider}")


__all__ = [
    "BedrockProvider",
    "EmbeddingProvider",
    "FakeEmbedder",
    "FakeProvider",
    "GeminiEmbedder",
    "GeminiProvider",
    "LLMProvider",
    "ObservedProvider",
    "TitanEmbedder",
    "build_embedding_provider",
    "build_llm_provider",
]
