import os
import shutil
import socket
import subprocess
import sys
from dataclasses import replace
from datetime import UTC, datetime, timedelta
from threading import Event
from uuid import uuid4

import httpx
import jwt
import psycopg
import pytest
from fastapi.testclient import TestClient
from identity_fixture import ADMIN, ISSUER, KEY, PUBLIC, identity
from psycopg import sql
from psycopg.conninfo import make_conninfo
from test_analytics import contract
from test_supply import SKU, WAREHOUSE, encoded, manifest

from analytics_service import db, snapshots
from analytics_service.api import PREFIX, PilotAuth, create_app
from analytics_service.config import AuthSettings, Settings
from analytics_service.events import TZ
from analytics_service.migrate import migrate

pytestmark = [
    pytest.mark.integration,
    pytest.mark.skipif(
        os.getenv("RUN_ANALYTICS_INTEGRATION") != "1", reason="Requires Compose test stack"
    ),
]


def cli(*args, **env):
    result = subprocess.run(
        [sys.executable, "-m", "analytics_service", *map(str, args)],
        env={**os.environ, **env},
        capture_output=True,
        text=True,
        timeout=20,
    )
    assert result.returncode == 0, result.stderr
    return result


@pytest.fixture
def supply_database(monkeypatch):
    admin_dsn = os.environ["ANALYTICS_PILOT_PG_DSN"]
    name = "supply_" + uuid4().hex
    database = sql.Identifier(name)
    with psycopg.connect(admin_dsn, autocommit=True) as admin:
        admin.execute(sql.SQL("CREATE DATABASE {}").format(database))
        try:
            monkeypatch.setenv("ANALYTICS_POSTGRES_DSN", make_conninfo(admin_dsn, dbname=name))
            monkeypatch.setenv("ANALYTICS_POSTGRES_API_ROLE", "analytics_api")
            monkeypatch.setenv("ANALYTICS_POSTGRES_WORKER_ROLE", "analytics_worker")
            monkeypatch.setenv("ANALYTICS_CLICKHOUSE_HOST", "")
            monkeypatch.setenv("ANALYTICS_SUPPLY_ENABLED", "1")
            yield Settings.load()
        finally:
            admin.execute(sql.SQL("DROP DATABASE {} WITH (FORCE)").format(database))


def migration_rows(settings):
    with db.connect(settings) as conn:
        return conn.execute(
            """SELECT store, version, checksum, xmin::text
               FROM public.analytics_schema_migration ORDER BY store, version"""
        ).fetchall()


def test_migrate_postgres_only_then_add_clickhouse(supply_database, tmp_path):
    settings = supply_database
    result = cli("migrate")
    assert "Skipping ClickHouse migrations: ANALYTICS_CLICKHOUSE_HOST is empty" in result.stderr
    postgres_rows = migration_rows(settings)
    assert [(row["store"], row["version"]) for row in postgres_rows] == [
        ("postgres", path.name) for path in sorted((settings.migrations / "postgres").glob("*.sql"))
    ]
    cli("migrate")
    assert migration_rows(settings) == postgres_rows

    cli(
        "migrate",
        ANALYTICS_CLICKHOUSE_HOST="clickhouse",
        ANALYTICS_CLICKHOUSE_USER="analytics_migrator",
        ANALYTICS_CLICKHOUSE_PASSWORD="local-migrator",
    )
    rows = migration_rows(settings)
    assert [row for row in rows if row["store"] == "postgres"] == postgres_rows
    assert [row["version"] for row in rows if row["store"] == "clickhouse"] == [
        path.name for path in sorted((settings.migrations / "clickhouse").glob("*.sql"))
    ]

    migrations = shutil.copytree(settings.migrations, tmp_path / "migrations")
    changed = next((migrations / "clickhouse").glob("*.sql"))
    with changed.open("a", encoding="utf-8") as file:
        file.write("\n-- edited after migration\n")
    configured = replace(
        settings,
        migrations=migrations,
        clickhouse_host="clickhouse",
        clickhouse_user="analytics_migrator",
        clickhouse_password="local-migrator",
    )
    with pytest.raises(ValueError, match="Migration checksum changed"):
        migrate(configured)
    assert migration_rows(settings) == rows


@pytest.fixture
def staff(monkeypatch):
    monkeypatch.setenv("ANALYTICS_AUTH_MODE", "scoped")
    monkeypatch.setenv("ANALYTICS_JWT_ISSUER", ISSUER)
    monkeypatch.setenv("ANALYTICS_JWT_PUBLIC_KEY", PUBLIC.decode())
    monkeypatch.setenv("ANALYTICS_IDENTITY_URL", ISSUER)
    monkeypatch.setenv("ANALYTICS_PILOT_IDS", ADMIN)
    auth = PilotAuth(AuthSettings.load())
    auth.client.close()

    def respond(request):
        return httpx.Response(200, json=identity(authorization=request.headers["Authorization"]))

    with httpx.Client(transport=httpx.MockTransport(respond)) as client:
        auth.client = client
        token = jwt.encode(
            {
                "sub": "admin",
                "user_id": ADMIN,
                "iss": ISSUER,
                "aud": "omama-api",
                "exp": datetime.now(UTC) + timedelta(minutes=5),
                "permission_grants": [
                    "ANALYTICS_SUPPLY_VIEW#GLOBAL",
                    "ANALYTICS_DASHBOARD_VIEW#GLOBAL",
                ],
            },
            KEY,
            algorithm="RS256",
        )
        yield auth, {"Authorization": f"Bearer {token}"}


def as_api(settings):
    return replace(
        settings,
        postgres_dsn=make_conninfo(
            settings.postgres_dsn, user="analytics_api", password="local-api"
        ),
        clickhouse_user="analytics_api",
        clickhouse_password="local-api",
    )


def publish_generation(settings):
    generation = uuid4()
    with db.connect(settings) as conn:
        conn.execute(
            """INSERT INTO serving_generation (generation_id,scope_key,pipeline,state,created_at)
               VALUES (%s,%s,%s,'BUILDING',clock_timestamp())""",
            (generation, settings.scope, settings.pipeline),
        )
        assert snapshots.publish(
            conn, settings, generation, None, [], {"days": {}}, None, 0, bytes(32)
        )
    return generation


def test_supply_only_readiness_and_manual_import(supply_database, staff, tmp_path, monkeypatch):
    settings = supply_database
    cli("migrate")
    now = datetime.now(UTC)
    today = now.astimezone(TZ).date()
    day = today - timedelta(days=1)
    export = tmp_path / "order-day.json"
    export.write_bytes(encoded(manifest("order_day", day, now, soldUnits=10, returnedUnits=1)))
    cli("import-supply", export)
    generation = publish_generation(settings)

    def unexpected_connection(**kwargs):
        pytest.fail("Supply-only API attempted to connect to ClickHouse")

    monkeypatch.setattr("clickhouse_connect.get_client", unexpected_connection)
    auth, headers = staff
    query = {"from": str(day), "to": str(today)}
    with TestClient(create_app(as_api(settings), auth)) as client:
        assert client.get("/livez").json() == {"status": "ok"}
        ready = client.get("/readyz")
        assert ready.status_code == 200
        assert ready.json() == {
            "status": "ready",
            "capabilities": {"supply": "ready", "dashboard": "disabled"},
        }
        contract("AnalyticsReadiness", ready)
        workbench = client.get(
            PREFIX + "/supply/workbench",
            params={**query, "warehouseId": WAREHOUSE},
            headers=headers,
        )
        assert workbench.status_code == 200, workbench.text
        assert [row["skuCode"] for row in workbench.json()["ranking"]] == [SKU]
        for path, params in (
            ("/dashboard", query),
            ("/metrics/orders_paid/series", {**query, "snapshotId": str(generation)}),
        ):
            response = client.get(PREFIX + path, params=params, headers=headers)
            assert response.status_code == 503, response.text
            assert response.json()["code"] == "ANALYTICS_TEMPORARILY_UNAVAILABLE"
            contract("Error", response)


def test_configured_dashboard_readiness_requires_generation_and_clickhouse(
    supply_database, staff, monkeypatch
):
    settings = supply_database
    cli("migrate")
    configured = replace(as_api(settings), clickhouse_host="clickhouse")
    auth, headers = staff
    with TestClient(create_app(configured, auth)) as client:
        assert client.get("/readyz").json() == {
            "status": "degraded",
            "capabilities": {"supply": "ready", "dashboard": "not_ready"},
        }
        publish_generation(settings)
        assert client.get("/readyz").json() == {
            "status": "ready",
            "capabilities": {"supply": "ready", "dashboard": "ready"},
        }

    # Reserve a TCP port without listening, simulating a stopped server without touching the suite.
    with socket.socket() as stopped:
        stopped.bind(("127.0.0.1", 0))
        down = replace(
            configured, clickhouse_host="127.0.0.1", clickhouse_port=stopped.getsockname()[1]
        )
        with TestClient(create_app(down, auth)) as client:
            readiness = client.get("/readyz")
            assert readiness.status_code == 200, readiness.text
            assert readiness.json() == {
                "status": "degraded",
                "capabilities": {"supply": "ready", "dashboard": "not_ready"},
            }
            workbench = client.get(
                PREFIX + "/supply/workbench",
                params={"warehouseId": WAREHOUSE, "from": "2026-01-01", "to": "2026-01-02"},
                headers=headers,
            )
            assert workbench.status_code == 200, workbench.text

        with TestClient(create_app(replace(down, supply_enabled=False), auth)) as client:
            readiness = client.get("/readyz")
            assert readiness.status_code == 503, readiness.text
            assert readiness.json()["status"] == "not_ready"
            assert readiness.json()["capabilities"] == {
                "supply": "disabled",
                "dashboard": "not_ready",
            }
            contract("Error", readiness)

    release_probe = Event()

    def stalled_clickhouse(settings):
        if not release_probe.wait(5):
            pytest.fail("Readiness waited for the stalled ClickHouse probe")
        raise OSError("ClickHouse is unreachable")

    monkeypatch.setattr("analytics_service.api.clickhouse", stalled_clickhouse)
    with TestClient(create_app(configured, auth)) as client:
        try:
            readiness = client.get("/readyz")
            assert readiness.status_code == 200, readiness.text
            assert readiness.json() == {
                "status": "degraded",
                "capabilities": {"supply": "ready", "dashboard": "not_ready"},
            }
        finally:
            release_probe.set()
