"""Tool registrations for the shopping advisor.

Backends are the fixture implementations or the HTTP clients, selected by
configuration. Handlers enforce the tool-level business rules (only
purchasable products surface; order data only ever returns to its owner).
"""
from __future__ import annotations

import asyncio
from dataclasses import dataclass
from typing import Any

from pydantic import BaseModel, Field

from ...platform.gateway import Tool, ToolContext, ToolRegistry


@dataclass
class Backends:
    catalog: Any
    inventory: Any
    orders: Any
    shipping: Any
    analytics: Any


class SearchProductsArgs(BaseModel):
    query: str = ""
    max_price_vnd: int | None = None
    limit: int = Field(default=5, ge=1, le=6)


class CheckStockArgs(BaseModel):
    sku: str


class OrderStatusArgs(BaseModel):
    order_id: str


class ShippingArgs(BaseModel):
    province: str
    weight_grams: int = Field(default=500, ge=1, le=100_000)


class RecommendationsArgs(BaseModel):
    placement: str  # product_detail | cart | home
    product_id: str | None = None
    sku_codes: list[str] = Field(default_factory=list)
    limit: int = Field(default=4, ge=1, le=6)


def build_registry(backends: Backends) -> ToolRegistry:
    registry = ToolRegistry()

    async def search_products(ctx: ToolContext, args: SearchProductsArgs) -> dict:
        products = await asyncio.to_thread(
            backends.catalog.search, args.query, args.max_price_vnd, args.limit
        )
        return {"products": [p.public_dict() for p in products]}

    async def check_stock(ctx: ToolContext, args: CheckStockArgs) -> dict:
        available = await asyncio.to_thread(backends.inventory.check, args.sku)
        return {"sku": args.sku, "available": available}

    async def get_order_status(ctx: ToolContext, args: OrderStatusArgs) -> dict:
        order = await asyncio.to_thread(backends.orders.get, args.order_id, ctx.authorization)
        # Never leak existence: another customer's order looks exactly like
        # a non-existent one.
        if order is None or order.customer_id != ctx.customer_id:
            return {"found": False}
        return {
            "found": True,
            "order_id": order.id,
            "status": order.status,
            "summary": order.summary,
            "eta": order.eta,
        }

    async def estimate_shipping(ctx: ToolContext, args: ShippingArgs) -> dict:
        quote = await asyncio.to_thread(
            backends.shipping.estimate, args.province, args.weight_grams
        )
        return {"fee_vnd": quote.fee_vnd, "eta_days": quote.eta_days, "carrier": quote.carrier}

    async def get_recommendations(ctx: ToolContext, args: RecommendationsArgs) -> dict:
        context: dict = {}
        if args.placement == "product_detail" and args.product_id:
            context["productId"] = args.product_id
        if args.placement == "cart":
            context["skuCodes"] = args.sku_codes
        result = await asyncio.to_thread(
            backends.analytics.recommended,
            args.placement,
            context,
            args.limit,
            ctx.authorization,
        )
        return {"items": result.get("items", []), "strategy": result.get("strategy", "")}

    registry.register(Tool("search_products", "Search purchasable public products",
                           SearchProductsArgs, search_products))
    registry.register(Tool("check_stock", "Check real stock for a SKU",
                           CheckStockArgs, check_stock))
    registry.register(Tool("get_order_status", "Order status of the caller's own order",
                           OrderStatusArgs, get_order_status, requires_customer=True))
    registry.register(Tool("estimate_shipping", "Estimate shipping fee",
                           ShippingArgs, estimate_shipping))
    registry.register(Tool("get_recommendations", "Analytics-backed product recommendations",
                           RecommendationsArgs, get_recommendations))
    return registry
