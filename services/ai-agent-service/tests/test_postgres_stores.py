"""Integration tests against real PostgreSQL + pgvector.

Runs inside compose.test.yml (database service provided). Skipped when no
database is reachable, so the suite also passes on a bare checkout.
"""
import os
import uuid
from datetime import UTC, datetime, timedelta

import pytest

from ai_agent_service import db
from ai_agent_service.platform.conversations import PostgresConversationStore
from ai_agent_service.platform.knowledge import PostgresKnowledgeStore
from ai_agent_service.providers.embeddings import FakeEmbedder

DATABASE_URL = os.environ.get("AI_DATABASE_URL", "")
MIGRATIONS_DIR = os.environ.get("AI_MIGRATIONS_DIR", "migrations/postgres")

pytestmark = [
    pytest.mark.integration,
    pytest.mark.skipif(
        not DATABASE_URL or not db.ping(DATABASE_URL),
        reason="PostgreSQL not available",
    ),
]


@pytest.fixture(scope="module", autouse=True)
def _migrated():
    db.run_migrations(DATABASE_URL, MIGRATIONS_DIR)


def test_postgres_conversation_roundtrip_and_cleanup():
    store = PostgresConversationStore(DATABASE_URL)
    customer = f"cust-{uuid.uuid4().hex[:8]}"
    conv = store.create("customer", customer, 30)
    store.add_message(conv.id, "user", "xin chào")
    store.add_message(conv.id, "assistant", "chào anh/chị")
    messages = store.list_messages(conv.id)
    assert [m.role for m in messages] == ["user", "assistant"]

    store.set_slots(conv.id, {"budget_vnd": 200_000})
    assert store.get_slots(conv.id)["budget_vnd"] == 200_000

    assert store.get_history_consent(customer) is False
    store.set_history_consent(customer, True)
    assert store.get_history_consent(customer) is True

    expired = store.create("guest", None, 0)
    deleted = store.cleanup_expired(datetime.now(UTC) + timedelta(days=1))
    assert deleted >= 1
    assert store.get(expired.id) is None


@pytest.mark.asyncio
async def test_postgres_hybrid_search_and_deactivate():
    store = PostgresKnowledgeStore(DATABASE_URL, FakeEmbedder(1024), "fake-test")
    product_id = f"p-test-{uuid.uuid4().hex[:8]}"
    text = "Trà sen đặc biệt XYZ ướp hương sen, pha với nước 80 độ C."
    await store.ingest("Trà sen đặc biệt XYZ", "integration-test", text, product_id=product_id)
    results = await store.search(text, 3)
    assert any(r.product_id == product_id for r in results)

    removed = await store.deactivate_product(product_id)
    assert removed >= 1
    results = await store.search(text, 3)
    assert all(r.product_id != product_id for r in results)
