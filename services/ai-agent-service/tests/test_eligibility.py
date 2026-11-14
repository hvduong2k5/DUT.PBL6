"""Unified eligibility acceptance tests (docs ch. 12/13).

Unit tests pin the layer's semantics with fake stock checkers; service
and workflow tests pin that every Product Card path — shopping, basket,
recommendation placements and the popular fallback — actually passes
through it.
"""
import asyncio
import time

import httpx
import pytest
from fastapi.testclient import TestClient
from test_shopping_flow import cards, chat, create_conversation, done

from ai_agent_service.config import Settings
from ai_agent_service.integrations.fixtures import (
    FixtureAnalytics,
    FixtureCatalog,
    FixtureInventory,
    FixtureOrders,
    FixtureShipping,
)
from ai_agent_service.integrations.models import Product
from ai_agent_service.main import create_app
from ai_agent_service.modules.recommendation.service import RecommendationService
from ai_agent_service.modules.shopping_advisor.tools import Backends, build_registry
from ai_agent_service.platform.auth import Principal
from ai_agent_service.platform.eligibility import (
    INVENTORY_TIMEOUT,
    INVENTORY_UNKNOWN,
    OUT_OF_STOCK,
    OVER_BUDGET,
    ProductEligibility,
)
from ai_agent_service.platform.gateway import ToolGateway


def product(sku: str, price: int = 50_000, status: str = "published") -> Product:
    return Product(id=f"id-{sku}", sku=sku, name=sku, price_vnd=price, status=status)


def outcomes(result) -> dict[str, str]:
    return {r.sku: r.outcome for r in result.rejections}


# -- layer semantics -----------------------------------------------------------


async def test_budget_boundary_is_a_hard_cap():
    checked: list[str] = []

    async def check(sku: str) -> int:
        checked.append(sku)
        return 5

    layer = ProductEligibility()
    result = await layer.filter(
        [product("SKU-EQ", 100_000), product("SKU-OVER", 100_001)],
        budget_vnd=100_000, check_stock=check, source="test",
    )
    assert [e.product.sku for e in result.eligible] == ["SKU-EQ"]
    assert outcomes(result) == {"SKU-OVER": OVER_BUDGET}
    assert checked == ["SKU-EQ"]  # over-budget SKUs never reach Inventory


async def test_stock_outcomes_distinguish_verified_from_unverified():
    async def check(sku: str):
        if sku == "SKU-ZERO":
            return 0
        if sku == "SKU-ONE":
            return 1
        if sku == "SKU-NONE":
            return None
        if sku == "SKU-SLOW":
            raise TimeoutError("gateway timeout")
        raise RuntimeError("connection reset")

    layer = ProductEligibility()
    result = await layer.filter(
        [product(s) for s in ("SKU-ZERO", "SKU-ONE", "SKU-NONE", "SKU-SLOW", "SKU-ERR")],
        budget_vnd=None, check_stock=check, source="test",
    )
    assert [e.product.sku for e in result.eligible] == ["SKU-ONE"]
    assert outcomes(result) == {
        "SKU-ZERO": OUT_OF_STOCK,
        "SKU-NONE": INVENTORY_UNKNOWN,
        "SKU-SLOW": INVENTORY_TIMEOUT,
        "SKU-ERR": INVENTORY_UNKNOWN,
    }


async def test_unpublished_and_invalid_sku_never_reach_inventory():
    async def check(sku: str) -> int:
        raise AssertionError("checker must not be called")

    layer = ProductEligibility()
    result = await layer.filter(
        [product("SKU-OLD", status="discontinued"), product("")],
        budget_vnd=None, check_stock=check, source="test",
    )
    assert result.eligible == []
    assert outcomes(result)["SKU-OLD"] == "NOT_PUBLISHED"
    assert [r.outcome for r in result.rejections if r.sku == "id-"] == ["INVALID_SKU"]


async def test_dedupe_by_sku_checks_once():
    calls: list[str] = []

    async def check(sku: str) -> int:
        calls.append(sku)
        return 3

    layer = ProductEligibility()
    result = await layer.filter(
        [product("SKU-DUP"), product("SKU-DUP")],
        budget_vnd=None, check_stock=check, source="test",
    )
    assert len(result.eligible) == 1
    assert calls == ["SKU-DUP"]


async def test_partial_failure_keeps_verified_products():
    async def check(sku: str) -> int:
        if sku == "SKU-BAD":
            raise TimeoutError("slow inventory")
        return 2

    layer = ProductEligibility()
    skus = [f"SKU-{i}" for i in range(5)] + ["SKU-BAD"]
    result = await layer.filter(
        [product(s) for s in skus], budget_vnd=None, check_stock=check, source="test"
    )
    assert len(result.eligible) == 5
    assert outcomes(result) == {"SKU-BAD": INVENTORY_TIMEOUT}


async def test_stock_checks_run_in_parallel_with_a_bound():
    in_flight = 0
    max_in_flight = 0

    async def check(sku: str) -> int:
        nonlocal in_flight, max_in_flight
        in_flight += 1
        max_in_flight = max(max_in_flight, in_flight)
        await asyncio.sleep(0.15)
        in_flight -= 1
        return 1

    layer = ProductEligibility(concurrency=3, deadline_seconds=5)
    started = time.monotonic()
    result = await layer.filter(
        [product(f"SKU-{i}") for i in range(6)],
        budget_vnd=None, check_stock=check, source="test",
    )
    elapsed = time.monotonic() - started
    assert len(result.eligible) == 6
    assert 1 < max_in_flight <= 3  # parallel, but bounded by the semaphore
    assert elapsed < 0.6  # sequential would take ~0.9s


async def test_global_deadline_cuts_off_hanging_checks():
    async def check(sku: str) -> int:
        await asyncio.sleep(30)
        return 1

    layer = ProductEligibility(concurrency=3, deadline_seconds=0.2)
    started = time.monotonic()
    result = await layer.filter(
        [product("SKU-A"), product("SKU-B")],
        budget_vnd=None, check_stock=check, source="test",
    )
    elapsed = time.monotonic() - started
    assert elapsed < 2
    assert result.eligible == []
    assert outcomes(result) == {"SKU-A": INVENTORY_TIMEOUT, "SKU-B": INVENTORY_TIMEOUT}


# -- recommendation service integration -----------------------------------------

def make_service(analytics=None) -> RecommendationService:
    catalog = FixtureCatalog()
    backends = Backends(
        catalog=catalog,
        inventory=FixtureInventory(),
        orders=FixtureOrders(),
        shipping=FixtureShipping(),
        analytics=analytics or FixtureAnalytics(catalog),
    )
    gateway = ToolGateway(build_registry(backends), timeout_seconds=2.0)
    return RecommendationService(backends, gateway)


async def test_related_excludes_out_of_stock():
    service = make_service()
    products = await service.related("p-deo")
    # Analytics fixture suggests p-gion (stock 0) and p-gung (stock 8):
    # only the verified in-stock product may surface now.
    assert [p.id for p in products] == ["p-gung"]


async def test_home_excludes_out_of_stock():
    service = make_service()
    products = await service.home(None)
    ids = {p.id for p in products}
    assert products
    assert "p-gion" not in ids  # popularity never resurrects an OOS product


async def test_basket_applies_session_budget():
    service = make_service()
    products = await service.basket(["MX-GIFT-320"], budget_vnd=46_000)
    # Trà sen 49.000đ is over the session budget; mứt gừng 45.000đ is not.
    assert [p.id for p in products] == ["p-gung"]


class _StubAnalytics:
    def __init__(self, result=None, error=None):
        self.result = result
        self.error = error

    def recommended(self, placement, context, limit=6, authorization=None):
        if self.error is not None:
            raise self.error
        return self.result


async def test_fallback_popular_is_eligibility_filtered_too():
    stub = _StubAnalytics(result={"items": [{"productId": "p-gion"}], "strategy": "stub"})
    service = make_service(analytics=stub)
    products = await service.basket(["MX-DEO-180"])
    ids = {p.id for p in products}
    assert "p-gion" not in ids  # the only Analytics item is out of stock
    assert "p-deo" in ids  # popular fallback took over, also filtered


async def test_analytics_timeout_falls_back_without_breaking():
    stub = _StubAnalytics(error=httpx.ReadTimeout("slow", request=httpx.Request("GET", "http://x")))
    service = make_service(analytics=stub)
    products = await service.home(None)
    assert products
    assert "p-gion" not in {p.id for p in products}


# -- workflow integration ---------------------------------------------------------


def make_app():
    app = create_app(Settings())

    def fake_auth(header):
        if header and header.startswith("Bearer "):
            return Principal(customer_id=header[7:].strip(), is_guest=False)
        return Principal(customer_id=None, is_guest=True)

    app.state.authenticator = fake_auth
    return app


@pytest.fixture()
def client():
    with TestClient(make_app()) as test_client:
        yield test_client


def test_basket_chat_respects_session_budget(client):
    cid = create_conversation(client)
    first = chat(client, cid, "Muốn mua quà tặng mẹ")
    assert done(first)["intent"] == "clarification"
    second = chat(client, cid, "Ngân sách 46k, tặng mẹ")
    assert done(second)["intent"] == "shopping"
    third = chat(client, cid, "Mua kèm gì với MX-GIFT-320 thì hợp?")
    assert done(third)["intent"] == "basket"
    assert [c["id"] for c in cards(third)] == ["p-gung"]


def test_basket_chat_without_budget_keeps_in_stock_suggestions(client):
    cid = create_conversation(client)
    events = chat(client, cid, "Mua kèm gì với MX-GIFT-320 thì hợp?")
    assert done(events)["intent"] == "basket"
    assert {c["id"] for c in cards(events)} == {"p-tra", "p-gung"}
