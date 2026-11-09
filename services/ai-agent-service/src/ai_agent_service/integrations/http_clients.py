"""HTTP adapters to the real business services (used when AI_FIXTURES=0).

These map the services' public APIs onto the internal interfaces the tools
use. The internal interface is the contract the agent depends on; if a
service's public shape changes, only this file changes.
"""
from __future__ import annotations

import httpx

from .models import OrderInfo, Product, ShippingQuote


def _client(base_url: str) -> httpx.Client:
    return httpx.Client(base_url=base_url, timeout=1.8)


class HttpCatalog:
    def __init__(self, base_url: str):
        self.base_url = base_url

    def search(self, query: str = "", max_price_vnd: int | None = None, limit: int = 6):
        params: dict = {"limit": limit, "status": "PUBLISHED"}
        if query:
            params["search"] = query
        if max_price_vnd is not None:
            params["maxPrice"] = max_price_vnd
        with _client(self.base_url) as client:
            resp = client.get("/api/v1/products", params=params)
            resp.raise_for_status()
            data = resp.json()
        items = data.get("data", data) if isinstance(data, dict) else data
        products = []
        for item in items or []:
            status = str(item.get("status", "PUBLISHED")).lower()
            products.append(
                Product(
                    id=str(item.get("id")),
                    sku=str(item.get("skuCode") or item.get("sku") or ""),
                    name=str(item.get("name", "")),
                    price_vnd=int(item.get("price") or item.get("priceVnd") or 0),
                    status="published" if status == "published" else "discontinued",
                    description=str(item.get("description", "")),
                )
            )
        return [p for p in products if p.status == "published"]

    def get(self, product_id: str):
        with _client(self.base_url) as client:
            resp = client.get(f"/api/v1/products/{product_id}")
            if resp.status_code == 404:
                return None
            resp.raise_for_status()
            item = resp.json()
        item = item.get("data", item)
        return Product(
            id=str(item.get("id")),
            sku=str(item.get("skuCode") or item.get("sku") or ""),
            name=str(item.get("name", "")),
            price_vnd=int(item.get("price") or item.get("priceVnd") or 0),
            status="published",
            description=str(item.get("description", "")),
        )

    def popular(self, limit: int = 6):
        return self.search("", None, limit)


class HttpInventory:
    def __init__(self, base_url: str):
        self.base_url = base_url

    def check(self, sku: str) -> int:
        with _client(self.base_url) as client:
            resp = client.get(f"/api/v1/stock/{sku}")
            if resp.status_code == 404:
                return 0
            resp.raise_for_status()
            data = resp.json()
        data = data.get("data", data)
        return int(data.get("available", 0))


class HttpOrders:
    def __init__(self, base_url: str):
        self.base_url = base_url

    def get(self, order_id: str, authorization: str | None = None):
        headers = {"Authorization": authorization} if authorization else {}
        with _client(self.base_url) as client:
            resp = client.get(f"/api/v1/orders/{order_id}", headers=headers)
            if resp.status_code in (401, 403, 404):
                return None
            resp.raise_for_status()
            data = resp.json()
        data = data.get("data", data)
        return OrderInfo(
            id=str(data.get("id", order_id)),
            customer_id=str(data.get("customerId", "")),
            status=str(data.get("status", "")),
            summary=str(data.get("summary", "")),
            eta=data.get("eta"),
        )


class HttpShipping:
    def __init__(self, base_url: str):
        self.base_url = base_url

    def estimate(self, province: str, weight_grams: int) -> ShippingQuote:
        with _client(self.base_url) as client:
            resp = client.post(
                "/api/v1/shipping/estimate",
                json={"province": province, "weightGrams": weight_grams},
            )
            resp.raise_for_status()
            data = resp.json()
        data = data.get("data", data)
        return ShippingQuote(
            fee_vnd=int(data.get("feeVnd", 0)),
            eta_days=str(data.get("etaDays", "")),
            carrier=str(data.get("carrier", "")),
        )


class HttpAnalytics:
    """Client for the verified Analytics recommendation contract."""

    def __init__(self, base_url: str):
        self.base_url = base_url

    def recommended(
        self, placement: str, context: dict, limit: int = 6, authorization: str | None = None
    ) -> dict:
        params: dict = {"placement": placement, "limit": limit}
        if placement == "product_detail":
            params["productId"] = context["productId"]
            if context.get("skuCode"):
                params["skuCode"] = context["skuCode"]
        if placement == "cart":
            params["skuCodes"] = context.get("skuCodes", [])
        headers = {"Authorization": authorization} if authorization else {}
        with _client(self.base_url) as client:
            resp = client.get("/api/v1/products/recommended", params=params, headers=headers)
            resp.raise_for_status()
            return resp.json()
