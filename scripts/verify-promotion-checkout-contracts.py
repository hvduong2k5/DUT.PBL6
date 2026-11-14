"""Contract guards for Promotion's Order/web/Kong integration (MVP requirements.txt)."""
import copy
import json
from pathlib import Path
import re
import unittest

import yaml
from jsonschema import Draft202012Validator, FormatChecker
from openapi_schema_validator import OAS30Validator

ROOT = Path(__file__).resolve().parents[1]


class PromotionCheckoutContracts(unittest.TestCase):
    def test_sandbox_order_snapshot_preserves_discount_fields_and_strict_shape(self):
        spec = yaml.safe_load((ROOT / "docs/03_api_specs/order-service.sandbox-3.1.openapi.yaml").read_text(encoding="utf-8"))

        def expand(value):
            if isinstance(value, list):
                return [expand(item) for item in value]
            if not isinstance(value, dict):
                return value
            if "$ref" in value:
                return expand(spec["components"]["schemas"][value["$ref"].rsplit("/", 1)[1]])
            return {key: expand(child) for key, child in value.items()}

        # Both the quote endpoint and Order.snapshot serialize domain.Quote,
        # including zero discounts when the Promotion provider is disabled.
        quote = dict(catalog_snapshot_id="catalog-1", catalog_expires_at="2026-11-14T11:00:00Z",
                     shipping_quote_reference="shipping-1", quote_id="quote-1", cart_revision=1,
                     payment_method="VIETQR", items=[dict(sku_code="MX-GION-500G", quantity=1)],
                     address=dict(recipient_name="Synthetic Customer", phone="+84905123456",
                                  street="Test only", ward_code="HUE-01", province_code="75"),
                     subtotal=110000, shipping_fee=25000, final_amount=135000,
                     expires_at="2026-11-14T10:49:00Z", shipping_expires_at="2026-11-14T10:49:00Z",
                     merchandise_discount=0, shipping_discount=0)
        schemas = spec["components"]["schemas"]
        for name, schema in [("Quote", schemas["Quote"]), ("Order.snapshot", schemas["Order"]["properties"]["snapshot"])]:
            validator = OAS30Validator(expand(schema), format_checker=FormatChecker())
            with self.subTest(schema=name):
                validator.validate(quote)
                legacy = {k: v for k, v in quote.items() if k not in ("merchandise_discount", "shipping_discount")}
                validator.validate(legacy)
                for field in ("merchandise_discount", "shipping_discount"):
                    for invalid in (-1, "0"):
                        self.assertFalse(validator.is_valid({**quote, field: invalid}))
                self.assertFalse(validator.is_valid({**quote, "unexpected_discount": 1}))

    def test_snapshot_versions_and_voucher_shape(self):
        base = ROOT / "docs/03_api_specs"
        schema = json.loads((base / "order-snapshot.v1.schema.json").read_text(encoding="utf-8"))
        Draft202012Validator.check_schema(schema)
        validator = Draft202012Validator(schema, format_checker=FormatChecker())
        fixtures = json.loads((base / "order-snapshot.v1.examples.json").read_text(encoding="utf-8"))["cases"]
        for case in fixtures:
            validator.validate(case["snapshot"])
        voucher = copy.deepcopy(fixtures[0]["snapshot"])
        voucher["snapshot_schema_version"] = "1.1"
        voucher["policies"]["promotion_policy_version"] = "promotion-voucher-1"
        voucher["money"]["merchandise_discount"] = 1000
        voucher["money"]["final_amount"] -= 1000
        voucher["voucher"] = dict(code="WELCOME", discount_type="FIXED_AMOUNT", discount_amount=1000,
                                  policy_version="promotion-voucher-1", valid_until=voucher["catalog"]["valid_until"],
                                  hold_id="10000000-0000-4000-8000-000000000001",
                                  hold_expires_at=voucher["catalog"]["valid_until"])
        validator.validate(voucher)
        voucher["snapshot_schema_version"] = "1.0"
        self.assertTrue(list(validator.iter_errors(voucher)), "v1.0 must keep its original no-discount semantics")

    def test_gateway_exposes_only_the_intended_promotion_surfaces(self):
        gateway = yaml.safe_load((ROOT / "services/api-gateway/config/kong.yml").read_text(encoding="utf-8"))
        service = next(s for s in gateway["services"] if s["name"] == "promotion-service")
        self.assertEqual(service["url"], "http://promotion-service:8017")
        self.assertEqual(service["retries"], 0)
        routes = service["routes"]
        for path, method in [("/api/v1/promotions/vouchers", "GET"), ("/api/v1/loyalty/points", "GET"),
                             ("/api/v1/admin/coupons", "POST")]:
            self.assertTrue(any(method in r["methods"] and any(
                re.match(p[1:], path) if p.startswith("~") else path.startswith(p) for p in r["paths"]
            ) for r in routes), (path, method))
        self.assertFalse(any("internal" in p for r in routes for p in r["paths"]))

    def test_customer_stable_checkout_matches_the_provider_projection(self):
        base = ROOT / "docs/03_api_specs"
        stable = yaml.safe_load((base / "customer-stable.openapi.yaml").read_text(encoding="utf-8"))
        self.assertEqual(stable["info"]["version"], "2.0.0")
        public = yaml.safe_load((base / "mvp-customer.openapi.yaml").read_text(encoding="utf-8"))
        for path in ["/api/v1/checkout/quote", "/api/v1/checkout", "/api/v1/checkout-operations/{id}"]:
            method = "post" if path != "/api/v1/checkout-operations/{id}" else "get"
            actual, expected = stable["paths"][path][method], public["paths"][path][method]
            self.assertEqual(actual["operationId"], expected["operationId"])
            self.assertEqual(actual["security"], [{"bearerAuth": []}])
            self.assertEqual(set(actual["responses"]), set(expected["responses"]))
        quote = stable["components"]["schemas"]["MvpOrderQuote"]["properties"]
        self.assertIn("merchandise_discount", quote)
        self.assertIn("shipping_discount", quote)
        operation = stable["components"]["schemas"]["MvpOrderCheckoutOperation"]
        self.assertEqual(operation["properties"]["order_id"]["type"], ["string", "null"])


if __name__ == "__main__":
    unittest.main()
