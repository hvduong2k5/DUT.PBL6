from __future__ import annotations

import os
from dataclasses import dataclass


def _env(name: str, default: str | None = None) -> str | None:
    value = os.environ.get(name)
    return value if value not in (None, "") else default


def _int(name: str, default: int) -> int:
    raw = _env(name)
    return int(raw) if raw is not None else default


def _float(name: str, default: float) -> float:
    raw = _env(name)
    return float(raw) if raw is not None else default


def _bool(name: str, default: bool = False) -> bool:
    raw = _env(name)
    if raw is None:
        return default
    return raw.strip().lower() in ("1", "true", "yes", "on")


@dataclass(frozen=True)
class Settings:
    """Runtime configuration, all from environment (12-factor)."""

    database_url: str = "postgresql://ai_agent:ai_agent@localhost:54329/ai_db"
    migrations_dir: str = "migrations/postgres"

    # LLM / embeddings (provider abstraction; see providers/)
    llm_provider: str = "fake"  # fake | gemini | bedrock
    model_id: str = "amazon.nova-pro-v1:0"  # decided 2026-11-09 (production)
    gemini_api_key: str | None = None
    gemini_base_url: str = "https://generativelanguage.googleapis.com/v1beta/openai/"
    gemini_model: str = "gemini-2.0-flash"
    aws_region: str = "ap-southeast-1"
    embedding_provider: str = "fake"  # fake | gemini | titan
    embedding_model: str = "amazon.titan-embed-text-v2:0"
    embedding_dim: int = 1024

    # Business adapters (read-only; always via the Tool Gateway)
    fixtures_mode: bool = True
    catalog_base_url: str = "http://localhost:8005"
    inventory_base_url: str = "http://localhost:8001"
    order_base_url: str = "http://localhost:8004"
    shipping_base_url: str = "http://localhost:8012"
    analytics_base_url: str = "http://localhost:8011"

    # Customer JWT verification (Keycloak / Identity)
    jwt_issuer: str = ""
    jwt_audience: str = ""
    jwks_url: str = ""

    # Agent execution limits (controlled runtime)
    tool_timeout_seconds: float = 2.0
    max_steps: int = 8
    max_tool_calls: int = 10
    max_reply_tokens: int = 700

    # Unified product eligibility (docs ch. 13): bounded parallel stock
    # checks under one global deadline per eligibility pass.
    eligibility_concurrency: int = 3
    eligibility_deadline_seconds: float = 1.8

    # Conversation retention (decided 2026-11-09: 30 / 15 days)
    retention_customer_days: int = 30
    retention_guest_days: int = 15

    @classmethod
    def load(cls) -> "Settings":
        return cls(
            database_url=_env(
                "AI_DATABASE_URL", "postgresql://ai_agent:ai_agent@localhost:54329/ai_db"
            ),
            migrations_dir=_env("AI_MIGRATIONS_DIR", "migrations/postgres"),
            llm_provider=_env("AI_LLM_PROVIDER", "fake"),
            model_id=_env("AI_MODEL_ID", "amazon.nova-pro-v1:0"),
            gemini_api_key=_env("GEMINI_API_KEY"),
            gemini_base_url=_env(
                "AI_GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/openai/"
            ),
            gemini_model=_env("AI_GEMINI_MODEL", "gemini-2.0-flash"),
            aws_region=_env("AWS_REGION", "ap-southeast-1"),
            embedding_provider=_env("AI_EMBEDDING_PROVIDER", "fake"),
            embedding_model=_env("AI_EMBEDDING_MODEL", "amazon.titan-embed-text-v2:0"),
            embedding_dim=_int("AI_EMBEDDING_DIM", 1024),
            fixtures_mode=_bool("AI_FIXTURES", True),
            catalog_base_url=_env("AI_CATALOG_BASE_URL", "http://localhost:8005"),
            inventory_base_url=_env("AI_INVENTORY_BASE_URL", "http://localhost:8001"),
            order_base_url=_env("AI_ORDER_BASE_URL", "http://localhost:8004"),
            shipping_base_url=_env("AI_SHIPPING_BASE_URL", "http://localhost:8012"),
            analytics_base_url=_env("AI_ANALYTICS_BASE_URL", "http://localhost:8011"),
            jwt_issuer=_env("AI_JWT_ISSUER", ""),
            jwt_audience=_env("AI_JWT_AUDIENCE", ""),
            jwks_url=_env("AI_JWKS_URL", ""),
            tool_timeout_seconds=_float("AI_TOOL_TIMEOUT_SECONDS", 2.0),
            max_steps=_int("AI_MAX_STEPS", 8),
            max_tool_calls=_int("AI_MAX_TOOL_CALLS", 10),
            max_reply_tokens=_int("AI_MAX_REPLY_TOKENS", 700),
            eligibility_concurrency=_int("AI_ELIGIBILITY_CONCURRENCY", 3),
            eligibility_deadline_seconds=_float("AI_ELIGIBILITY_DEADLINE_SECONDS", 1.8),
            retention_customer_days=_int("AI_RETENTION_CUSTOMER_DAYS", 30),
            retention_guest_days=_int("AI_RETENTION_GUEST_DAYS", 15),
        )
