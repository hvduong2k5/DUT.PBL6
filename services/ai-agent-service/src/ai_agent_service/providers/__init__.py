from __future__ import annotations

from ..config import Settings
from .embeddings import EmbeddingProvider, FakeEmbedder, GeminiEmbedder, TitanEmbedder
from .llm import BedrockProvider, FakeProvider, GeminiProvider, LLMProvider


def build_llm_provider(settings: Settings) -> LLMProvider:
    if settings.llm_provider == "gemini":
        return GeminiProvider(settings)
    if settings.llm_provider == "bedrock":
        return BedrockProvider(settings)
    return FakeProvider()


def build_embedding_provider(settings: Settings) -> EmbeddingProvider:
    if settings.embedding_provider == "gemini":
        return GeminiEmbedder(settings)
    if settings.embedding_provider == "titan":
        return TitanEmbedder(settings)
    return FakeEmbedder(settings.embedding_dim)


__all__ = [
    "BedrockProvider",
    "EmbeddingProvider",
    "FakeEmbedder",
    "FakeProvider",
    "GeminiEmbedder",
    "GeminiProvider",
    "LLMProvider",
    "TitanEmbedder",
    "build_embedding_provider",
    "build_llm_provider",
]
