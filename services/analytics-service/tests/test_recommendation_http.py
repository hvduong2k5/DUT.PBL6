import os

import httpx
import psycopg
import pytest

pytestmark = pytest.mark.skipif(
    os.getenv("RUN_ANALYTICS_INTEGRATION") != "1", reason="Requires isolated Compose runtimes"
)
BASE = "http://recommendation-api:8011"
IDENTITY = "http://identity-fixture:8016"
CONSENT = "/api/v1/recommendations/consent"


def test_customer_auth_over_actual_network():
    with httpx.Client(base_url=BASE, timeout=3) as client:
        assert client.get(CONSENT).status_code == 401
        for kind, expected in (("customer", 200), ("admin", 403), ("outage", 503)):
            token = httpx.get(IDENTITY + "/token/" + kind).json()["token"]
            result = client.get(CONSENT, headers={"Authorization": "Bearer " + token})
            assert result.status_code == expected, result.text
            assert result.headers["cache-control"] == "private, no-store"
        assert client.get(CONSENT, headers={"Authorization": "Bearer invalid"}).status_code == 401
        assert (
            client.get(
                CONSENT, headers={"X-User-ID": "fake", "X-User-Role": "CUSTOMER"}
            ).status_code
            == 401
        )
        cookie = "OMAMA_SESSION=f47ac10b-58cc-4372-a567-0e02b2c3d477"
        assert client.get(CONSENT, headers={"Cookie": cookie}).status_code == 200
        assert (
            client.put(
                CONSENT,
                headers={"Cookie": cookie, "Origin": "https://evil.test"},
                json={"enabled": True},
            ).status_code
            == 403
        )


def test_runtime_database_roles_enforce_publish_boundary():
    dsn = os.environ["ANALYTICS_PILOT_PG_DSN"]
    with psycopg.connect(dsn, dbname="analytics") as conn:
        for table in ("source_snapshot", "source_pointer", "generation", "serving_pointer"):
            name = "analytics_recommendation." + table
            assert conn.execute(
                "SELECT has_table_privilege('analytics_rec_worker',%s,'INSERT')", (name,)
            ).fetchone()[0]
            assert not conn.execute(
                "SELECT has_table_privilege('analytics_rec_api',%s,'INSERT')", (name,)
            ).fetchone()[0]
        assert conn.execute(
            "SELECT has_table_privilege('analytics_rec_api','analytics_recommendation.consent','UPDATE')"
        ).fetchone()[0]
