from __future__ import annotations

import asyncio
import logging

import httpx

from ...integrations.models import Product
from ...observability import RECOMMENDATION_OUTCOMES
from ...platform.eligibility import ProductEligibility, gateway_stock_checker
from ...platform.gateway import ToolContext
from ..shopping_advisor.tools import Backends

logger = logging.getLogger(__name__)

# Recommendation-level outcomes (docs ch. 13). NOT_CONSENTED is recorded
# when Analytics refuses the personalised fetch on consent grounds; the
# current contract usually answers with generic items instead, so the
# outcome exists for observability completeness.
SUCCESS = "SUCCESS"
EMPTY = "EMPTY"
NOT_CONSENTED = "NOT_CONSENTED"
TIMEOUT = "TIMEOUT"
UNAVAILABLE = "UNAVAILABLE"
INVALID_RESPONSE = "INVALID_RESPONSE"


class RecommendationService:
    """Consumes the MS-11 recommendation contract (docs ch. 11).

    Every product returned here — Analytics-backed or popular fallback —
    has passed the unified eligibility layer (docs ch. 13): resolved
    against the live Catalog, inside the caller's budget when one is
    known, and with stock verified available at check time.
    """

    def __init__(self, backends: Backends, gateway=None, eligibility=None):
        self.backends = backends
        self.gateway = gateway
        self.eligibility = eligibility or ProductEligibility()

    # -- stock checking ------------------------------------------------------
    async def _check_stock(self, sku: str, authorization: str | None) -> int:
        if self.gateway is not None:
            checker = gateway_stock_checker(
                self.gateway, ToolContext(authorization=authorization)
            )
            return await checker(sku)
        return await asyncio.wait_for(
            asyncio.to_thread(self.backends.inventory.check, sku),
            timeout=self.eligibility.deadline_seconds,
        )

    async def _eligible(
        self,
        products: list[Product],
        *,
        budget_vnd: int | None,
        authorization: str | None,
        source: str,
        limit: int,
    ) -> list[Product]:
        async def check(sku: str) -> int:
            return await self._check_stock(sku, authorization)

        result = await self.eligibility.filter(
            products, budget_vnd=budget_vnd, check_stock=check, source=source
        )
        return [entry.product for entry in result.eligible[:limit]]

    # -- Analytics fetch with outcome taxonomy --------------------------------
    async def _fetch(
        self, placement: str, context: dict, limit: int, authorization: str | None
    ) -> tuple[list[dict], str, str | None]:
        try:
            result = await asyncio.to_thread(
                self.backends.analytics.recommended, placement, context, limit, authorization
            )
        except httpx.TimeoutException:
            return [], TIMEOUT, None
        except httpx.HTTPStatusError as exc:
            if exc.response is not None and exc.response.status_code == 403:
                return [], NOT_CONSENTED, None
            return [], UNAVAILABLE, None
        except Exception:
            return [], UNAVAILABLE, None
        if not isinstance(result, dict) or not isinstance(result.get("items"), list):
            return [], INVALID_RESPONSE, None
        strategy = result.get("strategy")
        if not result["items"]:
            return [], EMPTY, strategy
        return result["items"], SUCCESS, strategy

    async def _resolve(self, items: list[dict]) -> list[Product]:
        products: list[Product] = []
        seen_ids: set[str] = set()
        for item in items:
            product_id = item.get("productId")
            if not product_id or product_id in seen_ids:
                continue  # missing id, or duplicate: deduplicate by product
            seen_ids.add(product_id)
            product = await asyncio.to_thread(self.backends.catalog.get, product_id)
            if product is not None:
                products.append(product)
        return products

    async def _from_analytics(
        self,
        placement: str,
        context: dict,
        limit: int,
        authorization: str | None,
        *,
        budget_vnd: int | None,
        source: str,
    ) -> list[Product]:
        items, outcome, strategy = await self._fetch(placement, context, limit, authorization)
        products: list[Product] = []
        if outcome == SUCCESS:
            products = await self._resolve(items)
            if not products:
                # Analytics answered, but nothing it named exists in the
                # Catalog: the response is unusable as recommendations.
                outcome = INVALID_RESPONSE
        RECOMMENDATION_OUTCOMES.labels(placement=placement, outcome=outcome).inc()
        # The strategy is logged for observability only. It is never turned
        # into a customer-facing explanation (docs ch. 13).
        logger.info(
            "recommendation placement=%s outcome=%s strategy=%s items=%d",
            placement, outcome, strategy, len(products),
        )
        if not products:
            return []
        return await self._eligible(
            products,
            budget_vnd=budget_vnd,
            authorization=authorization,
            source=source,
            limit=limit,
        )

    async def _popular(
        self,
        limit: int,
        *,
        budget_vnd: int | None = None,
        authorization: str | None = None,
        exclude_ids: set[str] | None = None,
    ) -> list[Product]:
        # Fetch extra candidates: eligibility may reject some of them.
        products = await asyncio.to_thread(self.backends.catalog.popular, limit + 4)
        exclude = exclude_ids or set()
        products = [p for p in products if p.id not in exclude]
        return await self._eligible(
            products,
            budget_vnd=budget_vnd,
            authorization=authorization,
            source="popular",
            limit=limit,
        )

    # -- placements ------------------------------------------------------------
    async def related(self, product_id: str, limit: int = 4) -> list[Product]:
        products = await self._from_analytics(
            "product_detail", {"productId": product_id}, limit, None,
            budget_vnd=None, source="recommendation",
        )
        return products or await self._popular(limit, exclude_ids={product_id})

    async def basket(
        self, sku_codes: list[str], limit: int = 4, budget_vnd: int | None = None
    ) -> list[Product]:
        products = await self._from_analytics(
            "cart", {"skuCodes": sku_codes}, limit, None,
            budget_vnd=budget_vnd, source="basket",
        )
        return products or await self._popular(limit, budget_vnd=budget_vnd)

    async def home(self, authorization: str | None, limit: int = 6) -> list[Product]:
        # Analytics owns the personalisation consent: with consent it returns
        # personalised items, without it generic ones. We never duplicate it.
        products = await self._from_analytics(
            "home", {}, limit, authorization,
            budget_vnd=None, source="recommendation",
        )
        return products or await self._popular(limit, authorization=authorization)
