from __future__ import annotations

import asyncio

from ...integrations.models import Product
from ..shopping_advisor.tools import Backends


class RecommendationService:
    def __init__(self, backends: Backends):
        self.backends = backends

    async def _resolve(self, items: list[dict], limit: int) -> list[Product]:
        products: list[Product] = []
        for item in items:
            product_id = item.get("productId")
            if not product_id:
                continue
            product = await asyncio.to_thread(self.backends.catalog.get, product_id)
            if product is not None and product.status == "published":
                products.append(product)
            if len(products) >= limit:
                break
        return products

    async def _from_analytics(
        self, placement: str, context: dict, limit: int, authorization: str | None
    ) -> list[Product]:
        try:
            result = await asyncio.to_thread(
                self.backends.analytics.recommended, placement, context, limit, authorization
            )
        except Exception:
            return []
        return await self._resolve(result.get("items", []), limit)

    async def _popular(self, limit: int, exclude_ids: set[str] | None = None) -> list[Product]:
        products = await asyncio.to_thread(self.backends.catalog.popular, limit + 2)
        exclude = exclude_ids or set()
        return [p for p in products if p.id not in exclude][:limit]

    async def related(self, product_id: str, limit: int = 4) -> list[Product]:
        products = await self._from_analytics(
            "product_detail", {"productId": product_id}, limit, None
        )
        return products or await self._popular(limit, exclude_ids={product_id})

    async def basket(self, sku_codes: list[str], limit: int = 4) -> list[Product]:
        products = await self._from_analytics("cart", {"skuCodes": sku_codes}, limit, None)
        return products or await self._popular(limit)

    async def home(self, authorization: str | None, limit: int = 6) -> list[Product]:
        # Analytics owns the personalisation consent: with consent it returns
        # personalised items, without it generic ones. We never duplicate it.
        products = await self._from_analytics("home", {}, limit, authorization)
        return products or await self._popular(limit)
