"""Fixture backends: in-repo sample data implementing the same interfaces
as the HTTP clients. Used by tests, CI and fixture-mode development so the
agent can be exercised end-to-end before every business service is wired.
"""
from __future__ import annotations

import re

from .models import OrderInfo, Product, ShippingQuote

_PRODUCTS = [
    Product("p-deo", "MX-DEO-180", "Mè xửng dẻo truyền thống 180g", 49_000, "published",
            "Mè xửng dẻo, vị ngọt truyền thống Huế, mềm thơm mè rang, hợp làm quà Tết.", 90),
    Product("p-gion", "MX-GION-160", "Mè xửng giòn 160g", 55_000, "published",
            "Mè xửng giòn rụm, ít ngọt hơn loại dẻo, đóng gói hút chân không.", 70),
    Product("p-den", "MX-DEN-180", "Mè xửng mè đen 180g", 59_000, "published",
            "Mè xửng mè đen thơm bùi, vị ngọt vừa, hộp giấy trang nhã.", 60),
    Product("p-gift", "MX-GIFT-320", "Hộp quà Chuyện Huế 320g", 169_000, "published",
            "Hộp quà gồm mè xửng dẻo, mè xửng giòn và trà sen, kèm thiệp viết tay.", 95),
    Product("p-tra", "TRA-SEN-20", "Trà sen Huế 20g", 49_000, "published",
            "Trà ướp sen hồ Tịnh Tâm, hương nhẹ, hợp tặng người lớn tuổi.", 50),
    Product("p-gung", "MG-GUNG-120", "Mứt gừng Huế 120g", 45_000, "published",
            "Mứt gừng cay nhẹ, ấm bụng, món quà nhỏ cho dịp lạnh và Tết.", 40),
    Product("p-old", "MX-OLD-100", "Mè xửng phiên bản mùa cũ 100g", 39_000, "discontinued",
            "Phiên bản giới hạn đã ngừng sản xuất.", 10),
]

_STOCK = {
    "MX-DEO-180": 25,
    "MX-GION-160": 0,  # out of stock (golden case)
    "MX-DEN-180": 12,
    "MX-GIFT-320": 5,
    "TRA-SEN-20": 40,
    "MG-GUNG-120": 8,
    "MX-OLD-100": 99,
}

_ORDERS = {
    "DH1001": OrderInfo("DH1001", "cust-1", "SHIPPED",
                        "1 x Hộp quà Chuyện Huế 320g", "ngày mai"),
    "DH1002": OrderInfo("DH1002", "cust-2", "DELIVERED",
                        "2 x Mè xửng dẻo truyền thống 180g", None),
}

_WORD_RE = re.compile(r"[a-zà-ỹ0-9]+", re.IGNORECASE)


def _tokens(text: str) -> set[str]:
    return {w.lower() for w in _WORD_RE.findall(text)}


class FixtureCatalog:
    def search(self, query: str = "", max_price_vnd: int | None = None, limit: int = 6):
        words = _tokens(query)
        scored = []
        for product in _PRODUCTS:
            if product.status != "published":
                continue  # tools only ever surface purchasable products
            if max_price_vnd is not None and product.price_vnd > max_price_vnd:
                continue
            haystack = _tokens(product.name + " " + product.description)
            overlap = len(words & haystack)
            scored.append((overlap, product.popularity, product))
        scored.sort(key=lambda item: (item[0], item[1]), reverse=True)
        return [p for _, _, p in scored[:limit]]

    def get(self, product_id: str):
        for product in _PRODUCTS:
            if product.id == product_id:
                return product
        return None

    def popular(self, limit: int = 6):
        items = [p for p in _PRODUCTS if p.status == "published"]
        items.sort(key=lambda p: p.popularity, reverse=True)
        return items[:limit]


class FixtureInventory:
    def check(self, sku: str) -> int:
        return _STOCK.get(sku, 0)


class FixtureOrders:
    def get(self, order_id: str, authorization: str | None = None):
        if not authorization or not authorization.startswith("Bearer "):
            return None
        customer_id = authorization[7:].strip()
        order = _ORDERS.get(order_id.upper())
        if order is None or order.customer_id != customer_id:
            return None
        return order


class FixtureShipping:
    def estimate(self, province: str, weight_grams: int) -> ShippingQuote:
        local = "huế" in province.lower() or "hue" in province.lower()
        fee = 22_000 if local else 32_000
        if weight_grams > 1000:
            fee += 5_000 * ((weight_grams - 1000 + 499) // 500)
        return ShippingQuote(fee_vnd=fee, eta_days="1-2" if local else "2-4")


class FixtureAnalytics:
    """Mirrors the verified Analytics recommendation contract (PR #62 branch)."""

    _RELATED = {"p-deo": ["p-gion", "p-gung"], "p-gift": ["p-tra", "p-deo"]}
    _BASKET = {"MX-DEO-180": ["TRA-SEN-20"], "MX-GIFT-320": ["TRA-SEN-20", "MG-GUNG-120"]}

    def __init__(self, catalog: FixtureCatalog):
        self.catalog = catalog

    def recommended(
        self, placement: str, context: dict, limit: int = 6, authorization: str | None = None
    ) -> dict:
        ids: list[str] = []
        if placement == "product_detail":
            ids = self._RELATED.get(context.get("productId", ""), [])
        elif placement == "cart":
            for sku in context.get("skuCodes", []):
                ids += self._BASKET.get(sku, [])
        elif placement == "home":
            ids = [p.id for p in self.catalog.popular(limit)]
        by_id = {p.id: p for p in _PRODUCTS}
        by_sku = {p.sku: p for p in _PRODUCTS}
        items = []
        for ref in ids:
            product = by_id.get(ref) or by_sku.get(ref)
            if product and product.status == "published":
                items.append({"productId": product.id, "skuCode": product.sku})
        return {"items": items[:limit], "strategy": "fixture"}
