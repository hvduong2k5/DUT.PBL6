from __future__ import annotations

from dataclasses import dataclass


@dataclass
class Product:
    id: str
    sku: str
    name: str
    price_vnd: int
    status: str  # published | discontinued
    description: str = ""
    popularity: int = 0

    def public_dict(self) -> dict:
        return {
            "id": self.id,
            "sku": self.sku,
            "name": self.name,
            "price_vnd": self.price_vnd,
            "description": self.description,
        }


@dataclass
class OrderInfo:
    id: str
    customer_id: str
    status: str
    summary: str
    eta: str | None = None


@dataclass
class ShippingQuote:
    fee_vnd: int
    eta_days: str
    carrier: str = "GHN"
