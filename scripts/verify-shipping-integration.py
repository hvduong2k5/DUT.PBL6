"""Shipping routing, IAM seed and deployment regression (no live-service claims)."""
from pathlib import Path
import re
import sqlite3
import unittest

import yaml
from jsonschema import Draft202012Validator, FormatChecker

ROOT = Path(__file__).resolve().parents[1]


class ShippingIntegrationTests(unittest.TestCase):
    def setUp(self):
        self.kong = yaml.safe_load((ROOT / "services/api-gateway/config/kong.yml").read_text(encoding="utf-8"))
        self.services = {s["name"]: s for s in self.kong["services"]}

    def matches(self, method, path):
        return [s["name"] for s in self.kong["services"] if s["name"].startswith("shipping-")
                for route in s["routes"] if method in route["methods"]
                for pattern in route["paths"] if re.match(pattern[1:], path)]

    def test_only_employee_reads_and_authenticated_ghn_callback_are_routed(self):
        self.assertEqual(self.matches("GET", "/api/v2/admin/shipping/shipments"), ["shipping-admin-service"])
        self.assertEqual(self.matches("GET", "/api/v2/admin/shipping/shipments/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"), ["shipping-admin-service"])
        self.assertEqual(self.matches("POST", "/api/v1/webhooks/logistics/ghn"), ["shipping-ghn-webhook-service"])
        for method, path in [
            ("POST", "/api/v1/internal/shipments"), ("POST", "/api/v1/internal/return-shipments"),
            ("GET", "/api/v1/orders/private-order/tracking"), ("POST", "/api/v1/webhooks/logistics/fake"),
            ("GET", "/api/v1/admin/shipping/workbench/shipments"), ("POST", "/api/v1/admin/shipping/shipments"),
            ("GET", "/api/v1/admin/shipping/shipments"), ("GET", "/api/v2/admin/shipping/workbench/shipments"),
            ("GET", "/api/v2/admin/shipping/shipments/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/cancel"),
            ("GET", "/api/v1/admin/shipping/shipments/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/cancel"),
            ("POST", "/api/v1/webhooks/logistics/ghn/extra"),
            ("GET", "/api/v2/admin/shipping/shipments/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/assignment"),
            ("POST", "/api/v2/admin/shipping/shipments/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/assignment"),
            ("GET", "/api/v2/admin/shipping/shipments/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/assignment/history"),
            ("POST", "/api/v2/admin/shipping/shipments/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/accept"),
            ("POST", "/api/v2/admin/shipping/shipments/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/progress"),
            ("GET", "/api/v2/admin/shipping/workbench/shipments/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"),
        ]:
            with self.subTest(method=method, path=path):
                self.assertEqual(self.matches(method, path), [])
        for name in ("shipping-admin-service", "shipping-ghn-webhook-service"):
            service = self.services[name]
            self.assertTrue(service["url"].startswith("http://shipping-service:8012"))
            self.assertEqual(service["retries"], 0)
            for route in service["routes"]:
                self.assertFalse(route["strip_path"])
                self.assertNotIn("plugins", route)  # Never inject a service token or bypass JWT.
        # Public v2 aliases are implemented by Shipping: keep the full path unchanged.
        service = self.services["shipping-admin-service"]
        for suffix in ("/shipments", "/shipments/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"):
            path = "/api/v2/admin/shipping" + suffix
            pattern = next(p for p in service["routes"][0]["paths"] if re.match(p[1:], path))
            self.assertEqual(re.match(pattern[1:], path).group(0), path)
            self.assertEqual(service["url"] + path, "http://shipping-service:8012/api/v2/admin/shipping" + suffix)
            self.assertNotIn("(?", pattern)
            self.assertNotRegex(pattern, r"\\[1-9]")
        remove = next(p for p in self.kong["plugins"] if p["name"] == "request-transformer")["config"]["remove"]["headers"]
        self.assertIn("X-Internal-Token", remove)
        self.assertNotIn("Authorization", remove)
        self.assertNotIn("X-GHN-Webhook-Secret", remove)

    def test_network_overlay_does_not_expose_data_services(self):
        overlay = yaml.safe_load((ROOT / "services/shipping-service/docker-compose.gateway.yml").read_text())
        self.assertEqual(set(overlay["services"]), {"shipping-service"})
        self.assertTrue(overlay["networks"]["gateway"]["external"])
        self.assertIn("shipping-service", overlay["services"]["shipping-service"]["networks"]["gateway"]["aliases"])
        compose = yaml.safe_load((ROOT / "services/shipping-service/docker-compose.yml").read_text())
        self.assertTrue(all(p.startswith("127.0.0.1:") for p in compose["services"]["shipping-service"]["ports"]))

    def test_assignment_contract_and_examples_preserve_actor_boundary(self):
        spec = yaml.safe_load((ROOT / "docs/03_api_specs/admin-stable.openapi.yaml").read_text(encoding="utf-8"))
        path = spec["paths"]["/api/v2/admin/shipping/shipments/{shipmentId}/assignment"]
        self.assertEqual(path["post"]["x-required-permissions"], ["SHIPMENT_ASSIGN", "USER_VIEW"])
        self.assertEqual(path["get"]["x-required-permissions"], ["SHIPMENT_ASSIGN"])
        self.assertEqual(path["post"]["x-feature-flag"], "SHIPPING_INTERNAL_ASSIGNMENTS_ENABLED")
        schema = path["post"]["requestBody"]["content"]["application/json"]["schema"]
        example = path["post"]["requestBody"]["content"]["application/json"]["example"]
        validator = Draft202012Validator(schema, format_checker=FormatChecker())
        validator.validate(example)
        for invalid in (
            {**example, "actor_id": "a0000000-0000-0000-0000-000000000009"},
            {k: v for k, v in example.items() if k != "expected_version"},
            {**example, "expected_version": -1},
            {**example, "expected_version": 1.5},
            {**example, "driver_id": "not-a-uuid"},
            {**example, "reason": "x" * 501},
        ):
            self.assertFalse(validator.is_valid(invalid), invalid)
        for operation in (path["get"], path["post"], spec["paths"]["/api/v2/admin/shipping/shipments/{shipmentId}/assignment/history"]["get"]):
            self.assertEqual(operation["security"], [{"bearerAuth": []}])
            self.assertEqual(operation["x-required-grant-scope"], "GLOBAL")
            self.assertTrue({"401", "403", "404", "503"}.issubset(operation["responses"]))
            response = operation["responses"]["200"]["content"]["application/json"]
            schema = {**response["schema"], "components": spec["components"]}
            validate = Draft202012Validator(schema, format_checker=FormatChecker())
            if "example" in response:
                validate.validate(response["example"])
            for value in response.get("examples", {}).values():
                validate.validate(value["value"])

    def test_acceptance_contract_binds_actor_to_jwt(self):
        spec = yaml.safe_load((ROOT / "docs/03_api_specs/admin-stable.openapi.yaml").read_text(encoding="utf-8"))
        op = spec["paths"]["/api/v2/admin/shipping/shipments/{shipmentId}/accept"]["post"]
        self.assertEqual(op["security"], [{"bearerAuth": []}])
        self.assertEqual(op["x-required-permissions"], ["SHIPMENT_EXECUTE"])
        self.assertEqual(op["x-required-grant-scope"], "GLOBAL")
        self.assertEqual(op["x-feature-flag"], "SHIPPING_INTERNAL_ASSIGNMENTS_ENABLED")
        request = op["requestBody"]["content"]["application/json"]
        validator = Draft202012Validator(request["schema"])
        validator.validate(request["example"])
        for invalid in ({}, {"expected_version": 0}, {"expected_version": 1.5}, {"expected_version": 1, "actor_id": "spoof"}, {"expected_version": 1, "driver_id": "spoof"}):
            self.assertFalse(validator.is_valid(invalid), invalid)
        response = op["responses"]["200"]["content"]["application/json"]
        Draft202012Validator(response["schema"], format_checker=FormatChecker()).validate(response["example"])
        self.assertTrue({"400", "401", "403", "404", "409", "422", "503"}.issubset(op["responses"]))

    def test_progress_contract_and_internal_event_compatibility(self):
        spec = yaml.safe_load((ROOT / "docs/03_api_specs/admin-stable.openapi.yaml").read_text(encoding="utf-8"))
        op = spec["paths"]["/api/v2/admin/shipping/shipments/{shipmentId}/progress"]["post"]
        self.assertEqual(op["security"], [{"bearerAuth": []}])
        self.assertEqual(op["x-required-permissions"], ["SHIPMENT_EXECUTE"])
        self.assertEqual(op["x-required-grant-scope"], "GLOBAL")
        self.assertEqual(op["x-feature-flag"], "SHIPPING_INTERNAL_ASSIGNMENTS_ENABLED")
        request = op["requestBody"]["content"]["application/json"]
        validate = Draft202012Validator(request["schema"])
        example = request["example"]
        validate.validate(example)
        for invalid in ({}, {**example, "actor_id": "spoof"}, {**example, "assignment_version": 0}, {**example, "expected_revision": 0}, {**example, "status": "HANDED_OVER"}, {**example, "status": "CANCELLED"}, {**example, "reason": "x" * 501}):
            self.assertFalse(validate.is_valid(invalid), invalid)
        response = op["responses"]["200"]["content"]["application/json"]
        Draft202012Validator(response["schema"], format_checker=FormatChecker()).validate(response["example"])
        import json
        for name, status in (("shipment_delivered", "DELIVERED"), ("shipment_delivery_failed", "DELIVERY_FAILED"), ("shipment_status_changed", "IN_TRANSIT"), ("shipment_returning", "RETURNING"), ("shipment_returned", "RETURNED")):
            schema = json.loads((ROOT / f"packages/events/schemas/shipping/v2/{name}.event.json").read_text())
            event = json.loads((ROOT / "packages/events/examples/shipping/v2/shipment_delivered.json").read_text())
            event["data"]["carrier_code"] = "INTERNAL"
            event["data"]["status"] = status
            event["type"] = "vn.omama.shipping.shipment." + name.removeprefix("shipment_").replace("_", "-") + ".v2"
            Draft202012Validator(schema, format_checker=FormatChecker()).validate(event)

    def test_driver_read_contract_preserves_owned_projection(self):
        spec = yaml.safe_load((ROOT / "docs/03_api_specs/admin-stable.openapi.yaml").read_text(encoding="utf-8"))
        for suffix in ("", "/{shipmentId}"):
            op = spec["paths"]["/api/v2/admin/shipping/workbench/shipments" + suffix]["get"]
            self.assertEqual(op["security"], [{"bearerAuth": []}])
            self.assertEqual(op["x-required-permissions"], ["SHIPMENT_EXECUTE"])
            self.assertEqual(op["x-feature-flag"], "SHIPPING_INTERNAL_ASSIGNMENTS_ENABLED")
            content = op["responses"]["200"]["content"]["application/json"]
            validator = Draft202012Validator({**content["schema"], "components": spec["components"]}, format_checker=FormatChecker())
            if "example" in content:
                validator.validate(content["example"])
            for example in content.get("examples", {}).values():
                validator.validate(example["value"])
            self.assertTrue({"400", "401", "403", "404", "503"}.issubset(op["responses"]))
        self.assertNotIn("recipient", spec["components"]["schemas"]["ShippingDriverSummary"]["properties"])
        self.assertIn("recipient", spec["components"]["schemas"]["ShippingDriverDetail"]["allOf"][1]["properties"])

    def test_internal_delivery_seed_is_additive_scoped_and_idempotent(self):
        sql = (ROOT / "services/identity-service/src/main/resources/db/migration/V24__seed_internal_shipping_permissions.sql").read_text()
        db = sqlite3.connect(":memory:")
        self.addCleanup(db.close)
        db.executescript("""
            CREATE TABLE iam_permissions(permission_code TEXT PRIMARY KEY, resource_code TEXT, action_code TEXT, risk_level TEXT, scope_required BOOLEAN, description TEXT);
            CREATE TABLE iam_roles(role_id TEXT PRIMARY KEY, role_code TEXT, is_enabled BOOLEAN, archived_at TEXT);
            CREATE TABLE iam_role_permissions(role_id TEXT, permission_code TEXT, PRIMARY KEY(role_id,permission_code));
            INSERT INTO iam_roles VALUES ('active','ADMIN',TRUE,NULL),('disabled','ADMIN',FALSE,NULL),('archived','ADMIN',TRUE,'2026-01-01'),('other','FINANCE',TRUE,NULL);
        """)
        db.executescript(sql)
        db.executescript(sql)
        self.assertEqual(db.execute("SELECT role_id,permission_code FROM iam_role_permissions ORDER BY permission_code").fetchall(), [("active", "SHIPMENT_ASSIGN"), ("active", "SHIPMENT_EXECUTE")])
        self.assertEqual(db.execute("SELECT scope_required FROM iam_permissions").fetchall(), [(1,), (1,)])
        self.assertNotIn("INSERT INTO iam_role_assignments", sql)

    def test_identity_relay_contract_is_explicit_and_cookie_compatible(self):
        spec = yaml.safe_load((ROOT / "docs/03_api_specs/admin-stable.openapi.yaml").read_text(encoding="utf-8"))
        relay = {p: item for p, item in spec["paths"].items() if p.startswith("/api/v1/admin/shipping-api/")}
        self.assertEqual(len(relay), 8)
        for item in relay.values():
            for method, op in item.items():
                if method not in ("get", "post"):
                    continue
                self.assertEqual(op["security"], [{"adminSession": []}, {"bearerAuth": []}])
                self.assertEqual(op["x-feature-flag"], "IDENTITY_SHIPPING_API_ENABLED")
                self.assertEqual(op["x-required-grant-scope"], "GLOBAL")
        for path, item in spec["paths"].items():
            if path.startswith("/api/admin/shipping/company/"):
                for op in item.values():
                    self.assertIn({"adminSession": []}, op["security"])

    def test_seed_is_idempotent_and_only_grants_active_admin(self):
        sql = (ROOT / "services/identity-service/src/main/resources/db/migration/V23__seed_shipping_permissions.sql").read_text()
        # Portable SQL regression, not PostgreSQL/Flyway runtime acceptance.
        db = sqlite3.connect(":memory:")
        self.addCleanup(db.close)
        db.executescript("""
            CREATE TABLE iam_permissions(permission_code TEXT PRIMARY KEY, resource_code TEXT, action_code TEXT, risk_level TEXT, scope_required BOOLEAN, description TEXT);
            CREATE TABLE iam_roles(role_id TEXT PRIMARY KEY, role_code TEXT, is_enabled BOOLEAN, archived_at TEXT);
            CREATE TABLE iam_role_permissions(role_id TEXT, permission_code TEXT, PRIMARY KEY(role_id,permission_code));
            INSERT INTO iam_roles VALUES ('active','ADMIN',TRUE,NULL),('disabled','ADMIN',FALSE,NULL),('archived','ADMIN',TRUE,'2026-01-01'),('other','FINANCE',TRUE,NULL);
        """)
        db.executescript(sql)
        db.executescript(sql)
        self.assertEqual(db.execute("SELECT role_id,permission_code FROM iam_role_permissions").fetchall(), [("active", "SHIPMENT_VIEW")])
        self.assertEqual(db.execute("SELECT scope_required FROM iam_permissions").fetchone(), (1,))
        self.assertNotIn("INSERT INTO iam_role_assignments", sql)

    def test_contract_keeps_global_jwt_and_distinguishes_backend_projection(self):
        spec = yaml.safe_load((ROOT / "docs/03_api_specs/admin-stable.openapi.yaml").read_text(encoding="utf-8"))
        for path in ("/api/v1/admin/shipping/shipments", "/api/v1/admin/shipping/shipments/{shipmentId}"):
            operation = spec["paths"][path]["get"]
            self.assertEqual(operation["x-required-permissions"], ["SHIPMENT_VIEW"])
            self.assertEqual(operation["x-required-grant-scope"], "GLOBAL")
            self.assertIn({"bearerAuth": []}, operation["security"])
        self.assertFalse(spec["x-shipping-provider-integration"]["web-bff-connected"])
        for path in ("/api/v2/admin/shipping/shipments", "/api/v2/admin/shipping/shipments/{shipmentId}"):
            operation = spec["paths"][path]["get"]
            self.assertEqual(operation["security"], [{"bearerAuth": []}])
            self.assertEqual(operation["x-required-grant-scope"], "GLOBAL")
        self.assertIn("page_size", spec["paths"]["/api/v2/admin/shipping/shipments"]["get"]["responses"]["200"]["content"]["application/json"]["schema"]["required"])
        self.assertIn("tracking_events", spec["paths"]["/api/v2/admin/shipping/shipments/{shipmentId}"]["get"]["responses"]["200"]["content"]["application/json"]["schema"]["required"])

    def test_provider_openapi_is_valid_not_just_parseable_yaml(self):
        from openapi_spec_validator import validate_spec
        spec = yaml.safe_load((ROOT / "docs/03_api_specs/shipping-service.openapi.yaml").read_text(encoding="utf-8"))
        validate_spec(spec)


if __name__ == "__main__":
    unittest.main()
