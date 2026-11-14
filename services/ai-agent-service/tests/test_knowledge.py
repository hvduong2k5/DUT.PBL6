import pytest

from ai_agent_service.integrations.fixtures import FixtureCatalog
from ai_agent_service.platform.knowledge import MemoryKnowledgeStore, chunk_text
from ai_agent_service.providers.embeddings import FakeEmbedder


def test_chunk_text_splits_long_input():
    text = "\n".join(f"Đoạn {i} " + "x" * 120 for i in range(6))
    chunks = chunk_text(text, size=300, overlap=40)
    assert len(chunks) >= 3
    assert all(len(c) <= 320 for c in chunks)


@pytest.mark.asyncio
async def test_ingest_search_and_deactivate():
    store = MemoryKnowledgeStore(FakeEmbedder(64))
    await store.ingest(
        "Trà sen Huế", "faq", "Trà sen ướp hương sen hồ Tịnh Tâm, pha nước 80 độ.",
        product_id="p-tra",
    )
    results = await store.search("Trà sen ướp hương sen hồ Tịnh Tâm, pha nước 80 độ.", 3)
    assert results and results[0].product_id == "p-tra"

    removed = await store.deactivate_product("p-tra")
    assert removed == 1
    results = await store.search("Trà sen ướp hương sen hồ Tịnh Tâm, pha nước 80 độ.", 3)
    assert all(r.product_id != "p-tra" for r in results)


def test_fixture_catalog_never_returns_discontinued():
    catalog = FixtureCatalog()
    for product in catalog.search("mè xửng", None, 10):
        assert product.status == "published"
    for product in catalog.popular(10):
        assert product.status == "published"
