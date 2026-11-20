"""Regression for the E2E caller, independent of Docker and live Identity."""
import ast
from pathlib import Path
import unittest
from unittest.mock import Mock

from mvp_fulfillment_checks import read_shipping_for_employee
from mvp_customer_checks import confirm_shipping_handover


class ShippingFixtureAuthTests(unittest.TestCase):
    def test_handover_uses_internal_token_and_stable_action_key(self):
        response = (200, {"status": "HANDED_OVER", "revision": 2}, {})
        request = Mock(return_value=response)
        shipment = "a0000000-0000-0000-0000-000000000001"
        for _ in range(2):
            self.assertEqual(confirm_shipping_handover(request, "http://shipping", shipment, "internal-token"), response)
        self.assertEqual(request.call_count, 2)
        for call in request.call_args_list:
            self.assertEqual(call.args, ("http://shipping/api/v1/internal/shipments/" + shipment + "/handover", {}))
            self.assertEqual(call.kwargs, {
                "token": "internal-token", "method": "POST",
                "headers": {"Idempotency-Key": "web-e2e-handover-" + shipment},
            })
        confirm_shipping_handover(request, "http://shipping", "another-shipment", "internal-token")
        self.assertNotEqual(request.call_args.kwargs["headers"], request.call_args_list[0].kwargs["headers"])

    def test_admin_projection_uses_employee_bearer_and_v2_route(self):
        response = (200, {"shipment": {"package_id": "package-1"}}, {})
        request = Mock(return_value=response)
        self.assertEqual(read_shipping_for_employee(request, "http://127.0.0.1:8012", "shipment-1", "employee-jwt"), response)
        request.assert_called_once_with("http://127.0.0.1:8012/api/v2/admin/shipping/shipments/shipment-1", token="employee-jwt")

    def test_auth_failure_is_not_retried_with_service_credentials(self):
        for status in (401, 403):
            with self.subTest(status=status):
                response = (status, {"code": "UNAUTHORIZED"}, {})
                request = Mock(return_value=response)
                self.assertEqual(read_shipping_for_employee(request, "http://shipping", "shipment-1", "invalid-jwt"), response)
                self.assertEqual(request.call_count, 1)

    def test_shipping_oidc_is_bound_to_the_fixture_keycloak(self):
        # Parse only this setup block: the full E2E script requires Python 3.12+.
        source = (Path(__file__).parent / "test-web-identity.py").read_text(encoding="utf-8")
        begin = source.index("        env.update(\n            SHIPPING_ENV=")
        end = source.index("        shipping_binary", begin)
        import textwrap
        tree = ast.parse(textwrap.dedent(source[begin:end]))
        keywords = {kw.arg: kw.value for kw in tree.body[0].value.keywords}
        self.assertEqual(ast.literal_eval(keywords["SHIPPING_OIDC_AUDIENCE"]), "omama-api")
        self.assertEqual(keywords["SHIPPING_OIDC_ISSUER"].id, "issuer")
        self.assertEqual(keywords["SHIPPING_OIDC_JWKS_URL"].left.id, "issuer")
        self.assertEqual(ast.literal_eval(keywords["SHIPPING_OIDC_JWKS_URL"].right), "/protocol/openid-connect/certs")


if __name__ == "__main__":
    unittest.main()
