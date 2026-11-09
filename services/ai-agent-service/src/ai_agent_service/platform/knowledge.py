"""Product knowledge: ingestion + hybrid retrieval (vector + keyword).

RAG supplies meaning (origin, characteristics, gift suitability, FAQ).
It is never the source of price or stock facts — those come from the
Catalog/Inventory tools.
"""
from __future__ import annotations

import hashlib
import math
import re
from dataclasses import dataclass, field
from typing import Protocol

from ..observability import RETRIEVAL_RESULTS
from ..providers.embeddings import EmbeddingProvider

_WORD_RE = re.compile(r"[a-zà-ỹ0-9]+", re.IGNORECASE)


def chunk_text(text: str, size: int = 500, overlap: int = 80) -> list[str]:
    paragraphs = [p.strip() for p in text.split("\n") if p.strip()]
    chunks: list[str] = []
    current = ""
    for para in paragraphs or [text.strip()]:
        if len(current) + len(para) + 1 <= size:
            current = (current + "\n" + para).strip()
            continue
        if current:
            chunks.append(current)
        while len(para) > size:
            chunks.append(para[:size])
            para = para[size - overlap:]
        current = para
    if current:
        chunks.append(current)
    return chunks


def _keywords(text: str) -> set[str]:
    return {w.lower() for w in _WORD_RE.findall(text)}


def _cosine(a: list[float], b: list[float]) -> float:
    dot = sum(x * y for x, y in zip(a, b))
    na = math.sqrt(sum(x * x for x in a)) or 1.0
    nb = math.sqrt(sum(x * x for x in b)) or 1.0
    return dot / (na * nb)


@dataclass
class KnowledgeChunk:
    content: str
    document_title: str
    product_id: str | None = None
    score: float = 0.0


@dataclass
class _StoredChunk:
    content: str
    document_title: str
    product_id: str | None
    embedding: list[float]
    keywords: set[str] = field(default_factory=set)
    is_active: bool = True


class KnowledgeStore(Protocol):
    async def ingest(
        self,
        title: str,
        source: str,
        text: str,
        product_id: str | None = None,
        visibility: str = "public",
    ) -> int: ...

    async def search(self, query: str, limit: int = 5) -> list[KnowledgeChunk]: ...

    async def deactivate_product(self, product_id: str) -> int: ...


class MemoryKnowledgeStore:
    def __init__(self, embedder: EmbeddingProvider):
        self.embedder = embedder
        self._chunks: list[_StoredChunk] = []

    async def ingest(self, title, source, text, product_id=None, visibility="public"):
        parts = chunk_text(text)
        vectors = await self.embedder.embed(parts)
        for part, vector in zip(parts, vectors):
            self._chunks.append(
                _StoredChunk(
                    content=part,
                    document_title=title,
                    product_id=product_id,
                    embedding=vector,
                    keywords=_keywords(part) | _keywords(title),
                )
            )
        return len(parts)

    async def search(self, query, limit=5):
        if not self._chunks:
            RETRIEVAL_RESULTS.observe(0)
            return []
        (query_vector,) = await self.embedder.embed([query])
        query_words = _keywords(query)
        results = []
        for chunk in self._chunks:
            if not chunk.is_active:
                continue
            vector_score = _cosine(query_vector, chunk.embedding)
            overlap = len(query_words & chunk.keywords) / max(1, len(query_words))
            score = 0.6 * vector_score + 0.4 * overlap
            if score > 0.05:
                results.append(
                    KnowledgeChunk(
                        content=chunk.content,
                        document_title=chunk.document_title,
                        product_id=chunk.product_id,
                        score=score,
                    )
                )
        results.sort(key=lambda r: r.score, reverse=True)
        RETRIEVAL_RESULTS.observe(len(results[:limit]))
        return results[:limit]

    async def deactivate_product(self, product_id):
        count = 0
        for chunk in self._chunks:
            if chunk.product_id == product_id and chunk.is_active:
                chunk.is_active = False
                count += 1
        return count


class PostgresKnowledgeStore:
    """pgvector + PostgreSQL FTS hybrid search with in-SQL filtering
    (visibility, active flag, effective window) — pre-filter, never post-filter."""

    def __init__(self, database_url: str, embedder: EmbeddingProvider, embedding_model: str):
        self.database_url = database_url
        self.embedder = embedder
        self.embedding_model = embedding_model

    def _conn(self):
        from ..db import connect

        return connect(self.database_url)

    async def ingest(self, title, source, text, product_id=None, visibility="public"):
        parts = chunk_text(text)
        vectors = await self.embedder.embed(parts)
        with self._conn() as conn:
            row = conn.execute(
                "INSERT INTO knowledge_documents (title, source, visibility) "
                "VALUES (%s, %s, %s) RETURNING id",
                (title, source, visibility),
            ).fetchone()
            document_id = row[0]
            for index, (part, vector) in enumerate(zip(parts, vectors)):
                conn.execute(
                    "INSERT INTO knowledge_chunks (document_id, product_id, chunk_index, "
                    "content, search_text, embedding, visibility, content_hash, "
                    "embedding_model) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)",
                    (
                        document_id,
                        product_id,
                        index,
                        part,
                        f"{title}\n{part}",
                        "[" + ",".join(str(x) for x in vector) + "]",
                        visibility,
                        hashlib.sha256(part.encode("utf-8")).hexdigest(),
                        self.embedding_model,
                    ),
                )
        return len(parts)

    async def search(self, query, limit=5):
        (query_vector,) = await self.embedder.embed([query])
        vector_literal = "[" + ",".join(str(x) for x in query_vector) + "]"
        with self._conn() as conn:
            rows = conn.execute(
                "SELECT c.content, d.title, c.product_id, "
                "  (1 - (c.embedding <=> %s::vector)) AS vector_score, "
                "  ts_rank(c.tsv, plainto_tsquery('simple', %s)) AS keyword_score "
                "FROM knowledge_chunks c "
                "JOIN knowledge_documents d ON d.id = c.document_id "
                "WHERE c.is_active AND d.status = 'active' AND c.visibility = 'public' "
                "  AND (c.effective_from IS NULL OR c.effective_from <= CURRENT_DATE) "
                "  AND (c.effective_until IS NULL OR c.effective_until >= CURRENT_DATE) "
                "ORDER BY (0.6 * (1 - (c.embedding <=> %s::vector)) "
                "          + 0.4 * ts_rank(c.tsv, plainto_tsquery('simple', %s))) DESC "
                "LIMIT %s",
                (vector_literal, query, vector_literal, query, limit),
            ).fetchall()
        results = [
            KnowledgeChunk(
                content=row[0],
                document_title=row[1],
                product_id=row[2],
                score=float(row[3]) * 0.6 + float(row[4]) * 0.4,
            )
            for row in rows
        ]
        RETRIEVAL_RESULTS.observe(len(results))
        return results

    async def deactivate_product(self, product_id):
        with self._conn() as conn:
            rows = conn.execute(
                "UPDATE knowledge_chunks SET is_active = FALSE "
                "WHERE product_id = %s AND is_active RETURNING id",
                (product_id,),
            ).fetchall()
        return len(rows)
