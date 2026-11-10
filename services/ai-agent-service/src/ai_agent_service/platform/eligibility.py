"""Unified product eligibility — the single convergence point for Product Cards.

Every card the service shows (shopping workflow, basket suggestions, the
recommendation API placements and the popular fallback) must pass this
layer: published -> hard budget cap -> verified available stock, checked
per SKU. Design record: docs ch. 13 (ADR-01..04).

Failure policy is fail-closed per SKU: a SKU whose stock cannot be
verified (timeout or error) is dropped, never shown, and never reported
as out of stock; the other SKUs are unaffected. Eligibility is advisory:
it observes stock at check time and reserves nothing — reservation and
finalisation belong to Order/Inventory at checkout.
"""
from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable
from dataclasses import dataclass, field

from ..integrations.models import Product
from ..observability import ELIGIBILITY_REJECTIONS
from .gateway import ToolContext, ToolError, ToolGateway

# Product-level outcomes (docs ch. 13).
ELIGIBLE = "ELIGIBLE"
NOT_PUBLISHED = "NOT_PUBLISHED"
OVER_BUDGET = "OVER_BUDGET"
OUT_OF_STOCK = "OUT_OF_STOCK"
INVENTORY_TIMEOUT = "INVENTORY_TIMEOUT"
INVENTORY_UNKNOWN = "INVENTORY_UNKNOWN"
PRODUCT_NOT_FOUND = "PRODUCT_NOT_FOUND"
INVALID_SKU = "INVALID_SKU"

# A stock checker returns the verified available quantity for one SKU.
# Contract: raise TimeoutError for a timed-out check; raise any other
# exception (or return None) when availability cannot be determined.
StockChecker = Callable[[str], Awaitable[int | None]]


@dataclass
class EligibleProduct:
    product: Product
    available: int


@dataclass
class Rejection:
    sku: str
    outcome: str


@dataclass
class EligibilityResult:
    eligible: list[EligibleProduct] = field(default_factory=list)
    rejections: list[Rejection] = field(default_factory=list)

    def rejected_products(self, candidates: list[Product], outcome: str) -> list[Product]:
        skus = {r.sku for r in self.rejections if r.outcome == outcome}
        return [p for p in candidates if p.sku in skus]


class ProductEligibility:
    def __init__(self, *, concurrency: int = 3, deadline_seconds: float = 1.8):
        self.concurrency = max(1, concurrency)
        self.deadline_seconds = deadline_seconds

    async def filter(
        self,
        products: list[Product],
        *,
        budget_vnd: int | None,
        check_stock: StockChecker,
        source: str,
        fatal: tuple[type[BaseException], ...] = (),
    ) -> EligibilityResult:
        result = EligibilityResult()
        seen: set[str] = set()
        candidates: list[Product] = []
        for product in products:
            if not product.sku:
                result.rejections.append(Rejection(product.id, INVALID_SKU))
                continue
            if product.sku in seen:
                continue  # deduplicate by SKU, keep the first occurrence
            seen.add(product.sku)
            if product.status != "published":
                result.rejections.append(Rejection(product.sku, NOT_PUBLISHED))
                continue
            if budget_vnd is not None and product.price_vnd > budget_vnd:
                result.rejections.append(Rejection(product.sku, OVER_BUDGET))
                continue
            candidates.append(product)
        if candidates:
            outcomes = await self._check_all(candidates, check_stock, fatal)
            for product, (outcome, available) in zip(candidates, outcomes, strict=True):
                if outcome == ELIGIBLE:
                    result.eligible.append(EligibleProduct(product, available))
                else:
                    result.rejections.append(Rejection(product.sku, outcome))
        for rejection in result.rejections:
            ELIGIBILITY_REJECTIONS.labels(source=source, outcome=rejection.outcome).inc()
        return result

    async def _check_all(
        self,
        candidates: list[Product],
        check_stock: StockChecker,
        fatal: tuple[type[BaseException], ...],
    ) -> list[tuple[str, int]]:
        semaphore = asyncio.Semaphore(self.concurrency)

        async def one(product: Product) -> tuple[str, int]:
            async with semaphore:
                try:
                    available = await check_stock(product.sku)
                except asyncio.CancelledError:
                    raise
                except fatal:
                    raise
                except TimeoutError:
                    return INVENTORY_TIMEOUT, 0
                except Exception:
                    return INVENTORY_UNKNOWN, 0
                if available is None:
                    return INVENTORY_UNKNOWN, 0
                if available <= 0:
                    return OUT_OF_STOCK, 0
                return ELIGIBLE, int(available)

        tasks = [asyncio.ensure_future(one(product)) for product in candidates]
        _, pending = await asyncio.wait(tasks, timeout=self.deadline_seconds)
        for task in pending:
            task.cancel()
        outcomes: list[tuple[str, int]] = []
        for task in tasks:
            if task in pending or task.cancelled():
                # The global deadline cut this check off: availability was
                # never verified, which the taxonomy records as a timeout.
                outcomes.append((INVENTORY_TIMEOUT, 0))
            else:
                outcomes.append(task.result())
        return outcomes


def gateway_stock_checker(
    gateway: ToolGateway, ctx: ToolContext, state=None
) -> StockChecker:
    """Adapt the check_stock tool to the StockChecker contract.

    Gateway timeouts surface as TimeoutError; any other tool failure is a
    plain exception (availability unknown). When a TurnState is given the
    calls also consume the turn's tool budget.
    """

    async def check(sku: str) -> int:
        try:
            if state is not None:
                result = await state.call_tool(gateway, "check_stock", {"sku": sku}, ctx)
            else:
                result = await gateway.execute("check_stock", {"sku": sku}, ctx)
        except ToolError as exc:
            if exc.code == "tool_timeout":
                raise TimeoutError(str(exc)) from exc
            raise RuntimeError(str(exc)) from exc
        return int(result["available"])

    return check
