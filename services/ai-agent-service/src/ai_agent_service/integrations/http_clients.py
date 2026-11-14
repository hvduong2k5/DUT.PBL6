"""HTTP adapters to the real business services (used when AI_FIXTURES=0).

These map the services' public APIs onto the internal interfaces the tools
use. The internal interface is the contract the agent depends on; if a
service's public shape changes, only this file changes.
"""
from __future__ import annotations

import uuid
from concurrent.futures import ThreadPoolExecutor

import httpx

from .models import OrderInfo, Product, ShippingQuote


def _client(base_url: str) -> httpx.Client:
    return httpx.Client(base_url=base_url, timeout=1.8)


def _money_vnd(value: object) -> int:
    if not isinstance(value, dict) or value.get("currency_code") != "VND":
        return 0
    return int(value.get("units", 0)) + int(value.get("nanos", 0)) // 1_000_000_000


class HttpCatalog:
    def __init__(self, base_url: str):
        self.base_url = base_url
        self._stock_by_sku: dict[str, int] = {}

    def _map_detail(self, item: dict) -> Product | None:
        variants = item.get("variants") if isinstance(item.get("variants"), list) else []
        for candidate in variants:
            candidate_sku = candidate.get("sku_code")
            candidate_stock = candidate.get("stock_available")
            if candidate_sku and candidate_stock is not None:
                self._stock_by_sku[str(candidate_sku)] = max(0, int(candidate_stock))
        valid = [
            variant for variant in variants
            if variant.get("sku_code") and _money_vnd(variant.get("price")) > 0
        ]
        if not valid:
            return None
        variant = min(valid, key=lambda candidate: _money_vnd(candidate.get("price")))
        stock = variant.get("stock_available")
        if stock is not None:
            self._stock_by_sku[str(variant["sku_code"])] = max(0, int(stock))
        return Product(
            id=str(item.get("product_id", "")),
            sku=str(variant["sku_code"]),
            name=str(item.get("name", "")),
            price_vnd=_money_vnd(variant.get("price")),
            status="published",
            description=str(item.get("description") or item.get("summary") or ""),
            stock_available=max(0, int(stock)) if stock is not None else None,
        )

    def _detail(self, client: httpx.Client, id_or_slug: str) -> Product | None:
        response = client.get(f"/api/v1/products/{id_or_slug}")
        if response.status_code == 404:
            return None
        response.raise_for_status()
        data = response.json()
        item = data.get("data", data) if isinstance(data, dict) else {}
        return self._map_detail(item)

    def search(self, query: str = "", max_price_vnd: int | None = None, limit: int = 6):
        fetch_size = min(10, limit + 4)
        path = "/api/v1/products/search" if query else "/api/v1/products"
        params: dict = {"page": 1, "page_size": fetch_size}
        if query:
            params["query"] = query
        elif max_price_vnd is not None:
            params["max_price"] = max_price_vnd
        with _client(self.base_url) as client:
            resp = client.get(path, params=params)
            resp.raise_for_status()
            data = resp.json()
            items = data.get("items", []) if isinstance(data, dict) else []
        identifiers = [
            str(item.get("product_id") or item.get("slug") or "")
            for item in items
            if item.get("product_id") or item.get("slug")
        ]
        with ThreadPoolExecutor(max_workers=max(1, len(identifiers))) as executor:
            resolved = list(executor.map(self.get, identifiers))
        products = [
            product for product in resolved
            if product is not None
            and (max_price_vnd is None or product.price_vnd <= max_price_vnd)
        ][:limit]
        return products

    def get(self, product_id: str):
        with _client(self.base_url) as client:
            return self._detail(client, product_id)

    def popular(self, limit: int = 6):
        return self.search("", None, limit)

    def stock_for_sku(self, sku: str) -> int:
        if sku in self._stock_by_sku:
            return self._stock_by_sku[sku]
        # Unknown projections fail closed. The normal search/get paths hydrate
        # every SKU that eligibility is allowed to check.
        return 0


class HttpInventory:
    def __init__(self, catalog: HttpCatalog):
        self.catalog = catalog

    def check(self, sku: str) -> int:
        return self.catalog.stock_for_sku(sku)


class HttpOrders:
    def __init__(self, base_url: str):
        self.base_url = base_url

    def get(self, order_id: str, authorization: str | None = None):
        headers = {"Authorization": authorization} if authorization else {}
        headers["Accept"] = "application/vnd.omama.customer-mvp+json"
        with _client(self.base_url) as client:
            resolved_id = order_id
            try:
                uuid.UUID(order_id)
            except ValueError:
                listing = client.get(
                    "/api/v1/orders", params={"limit": 100}, headers=headers
                )
                if listing.status_code in (401, 403):
                    return None
                listing.raise_for_status()
                list_data = listing.json()
                list_data = (
                    list_data.get("data", list_data)
                    if isinstance(list_data, dict)
                    else {}
                )
                match = next(
                    (
                        item for item in list_data.get("items", [])
                        if str(item.get("order_code") or item.get("code") or "").upper()
                        == order_id.upper()
                    ),
                    None,
                )
                if match is None:
                    return None
                resolved_id = str(match.get("order_id") or match.get("id") or "")
                if not resolved_id:
                    return None
            resp = client.get(f"/api/v1/orders/{resolved_id}", headers=headers)
            if resp.status_code in (401, 403, 404):
                return None
            resp.raise_for_status()
            data = resp.json()
        data = data.get("data", data)
        return OrderInfo(
            id=str(data.get("order_id") or data.get("id") or resolved_id),
            customer_id="",
            status=str(data.get("status", "")),
            summary=_order_summary(data),
            eta=data.get("sla_due_at"),
        )


class HttpShipping:
    def __init__(self, base_url: str):
        self.base_url = base_url

    def estimate(self, province: str, weight_grams: int) -> ShippingQuote:
        # Shipping's quote contract requires a complete recipient and parcel.
        # Province and weight alone must never be turned into a fabricated fee.
        return ShippingQuote(
            fee_vnd=0,
            eta_days="",
            carrier="",
            available=False,
            reason="full_checkout_address_required",
        )


class HttpAnalytics:
    """Client for the verified Analytics recommendation contract."""

    def __init__(self, base_url: str):
        self.base_url = base_url

    def recommended(
        self, placement: str, context: dict, limit: int = 6, authorization: str | None = None
    ) -> dict:
        params: list[tuple[str, object]] = [("placement", placement), ("limit", limit)]
        if placement == "product_detail":
            params.append(("productId", context["productId"]))
            if context.get("skuCode"):
                params.append(("skuCode", context["skuCode"]))
        if placement == "cart":
            params.extend(("skuCodes", sku) for sku in context.get("skuCodes", []))
        headers = {"Authorization": authorization} if authorization else {}
        with _client(self.base_url) as client:
            resp = client.get("/api/v1/products/recommended", params=params, headers=headers)
            resp.raise_for_status()
            return resp.json()


def _order_summary(data: dict) -> str:
    snapshot = data.get("snapshot") if isinstance(data.get("snapshot"), dict) else {}
    items = snapshot.get("items") if isinstance(snapshot.get("items"), list) else []
    names = [str(item.get("product_name")) for item in items if item.get("product_name")]
    order_code = str(data.get("order_code") or data.get("code") or "")
    if names:
        return f"{order_code}: " + ", ".join(names[:3])
    return order_code or "Order status is available."
