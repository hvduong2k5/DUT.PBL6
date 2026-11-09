from __future__ import annotations

import hashlib
import math
import random
from typing import Protocol

from ..config import Settings
from .llm import ProviderError


class EmbeddingProvider(Protocol):
    async def embed(self, texts: list[str]) -> list[list[float]]: ...


def _normalize(vector: list[float]) -> list[float]:
    norm = math.sqrt(sum(x * x for x in vector)) or 1.0
    return [x / norm for x in vector]


class FakeEmbedder:
    """Deterministic hashing embedder for tests/CI.

    The same text always yields the same vector; retrieval relevance in tests
    is carried by the keyword (FTS) half of hybrid search, exactly as designed.
    """

    def __init__(self, dim: int = 1024):
        self.dim = dim

    async def embed(self, texts: list[str]) -> list[list[float]]:
        vectors = []
        for text in texts:
            seed = int.from_bytes(hashlib.sha256(text.encode("utf-8")).digest()[:8], "big")
            rng = random.Random(seed)
            vectors.append(_normalize([rng.uniform(-1.0, 1.0) for _ in range(self.dim)]))
        return vectors


class GeminiEmbedder:
    def __init__(self, settings: Settings):
        self.settings = settings

    async def embed(self, texts: list[str]) -> list[list[float]]:
        if not self.settings.gemini_api_key:
            raise ProviderError("GEMINI_API_KEY is not configured")
        import httpx

        async with httpx.AsyncClient(
            base_url=self.settings.gemini_base_url,
            headers={"Authorization": f"Bearer {self.settings.gemini_api_key}"},
            timeout=30.0,
        ) as client:
            resp = await client.post(
                "/embeddings",
                json={
                    "model": self.settings.embedding_model,
                    "input": texts,
                    "dimensions": self.settings.embedding_dim,
                },
            )
            resp.raise_for_status()
            data = resp.json()
        return [item["embedding"] for item in data["data"]]


class TitanEmbedder:
    """Production embeddings via Bedrock (Titan Embeddings V2)."""

    def __init__(self, settings: Settings):
        self.settings = settings
        self._session = None

    async def embed(self, texts: list[str]) -> list[list[float]]:
        import json as jsonlib

        import aioboto3

        if self._session is None:
            self._session = aioboto3.Session()
        vectors = []
        async with self._session.client(
            "bedrock-runtime", region_name=self.settings.aws_region
        ) as client:
            for text in texts:
                resp = await client.invoke_model(
                    modelId=self.settings.embedding_model,
                    body=jsonlib.dumps(
                        {
                            "inputText": text,
                            "dimensions": self.settings.embedding_dim,
                            "normalize": True,
                        }
                    ),
                )
                payload = jsonlib.loads(await resp["body"].read())
                vectors.append(payload["embedding"])
        return vectors
