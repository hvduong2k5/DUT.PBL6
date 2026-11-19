"""Regression for the E2E caller, independent of Docker and live Identity."""
import ast
from pathlib import Path
import unittest
from unittest.mock import Mock

from mvp_fulfillment_checks import read_shipping_for_employee


class ShippingFixtureAuthTests(unittest.TestCase):
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
