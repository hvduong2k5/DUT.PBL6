import json
import os
import subprocess
import sys
import time
from datetime import UTC, datetime, timedelta
from datetime import time as wall_time
from pathlib import Path
from threading import Event, Thread
from uuid import uuid4

import httpx
import pytest
import yaml
from confluent_kafka import Producer, TopicPartition
from jsonschema import Draft202012Validator, FormatChecker
from openapi_spec_validator import validate

from analytics_service import db, kafka, report, snapshots
from analytics_service.api import APIError, date_range
from analytics_service.config import TOPIC, Settings
from analytics_service.events import TZ, InvalidRecord, OrderAdapter
from analytics_service.migrate import clickhouse

FIXTURES = Path("/app/fixtures/order/v3")
if not FIXTURES.exists():
    ROOT = Path(__file__).resolve().parents[3]
    FIXTURES = ROOT / "packages/events/examples/order/v3"
    SPEC = ROOT / "docs/03_api_specs/analytics-service.candidate.openapi.yaml"
else:
    SPEC = Path("/app/contracts/analytics.openapi.yaml")
API = "http://api:8011/api/v1/admin/analytics"
IDENTITY = "http://identity-fixture:8016"


def event(kind, order, occurred, event_id=None):
    result = json.loads((FIXTURES / f"{kind}.json").read_text(encoding="utf-8"))
    result["id"] = str(event_id or uuid4())
    result["subject"] = f"order:{order}"
    result["data"]["aggregate_id"] = str(order)
    result["time"] = occurred.isoformat()
    return result


def encoded(payload):
    return json.dumps(payload).encode()


@pytest.mark.parametrize("kind", ["paid", "completed"])
def test_adapter_and_key(kind):
    order = uuid4()
    day = datetime(2026, 11, 1, tzinfo=TZ)
    payload = event(kind, order, day)
    adapter = OrderAdapter(Settings.load().schema_dir)
    actual = adapter.decode(str(order).encode(), encoded(payload))
    assert actual.order_id == order and actual.milestone == kind.upper()
    assert actual.day == day.date()
    with pytest.raises(InvalidRecord, match="ROUTING_MISMATCH"):
        adapter.decode(b"wrong", encoded(payload))


@pytest.mark.parametrize(
    "args",
    [
        ("20261101", "20261102"),
        ("2026-11-02", "2026-11-01"),
        ("2026-11-01", "2027-02-01"),
        ("0001-01-01", "0001-01-02", True),
    ],
)
def test_date_errors(args):
    with pytest.raises(APIError) as error:
        date_range(*args)
    assert error.value.status == 400


def test_comparison_with_zero_and_non_fresh():
    start, end = date_range("2026-11-01", "2026-11-02")
    current = report.observation(5, "FRESH", [])
    previous = report.observation(0, "FRESH", [])
    result = report.comparison(current, previous, start, end)
    assert (result["absoluteChange"], result["percentChange"]) == ("5", None)
    current["quality"] = "PARTIAL"
    assert report.comparison(current, previous, start, end)["absoluteChange"] is None


def test_normalizes_timestamp_without_losing_nanosecond_identity():
    order = uuid4()
    adapter = OrderAdapter(Settings.load().schema_dir)
    payload = event("paid", order, datetime(2026, 11, 1, tzinfo=TZ))
    payload["time"] = "2026-10-31t17:00:00.000000123z"
    first = adapter.decode(str(order).encode(), encoded(payload))
    payload["time"] = "2026-11-01T00:00:00.000000124+07:00"
    second = adapter.decode(str(order).encode(), encoded(payload))
    assert first.day == second.day
    assert (first.submicro_ns, second.submicro_ns) == (123, 124)
    assert first.normalized_digest != second.normalized_digest


def test_spec():
    validate(yaml.safe_load(SPEC.read_text(encoding="utf-8")))


def wait_for(function, timeout=30):
    until = time.monotonic() + timeout
    while time.monotonic() < until:
        result = function()
        if result:
            return result
        time.sleep(0.25)
    raise AssertionError(f"Timed out waiting for {function.__name__}")


def current_offsets(settings):
    with db.connect(settings) as conn:
        return {
            row["partition_id"]: row["next_offset"] for row in db.cursors(conn, settings.pipeline)
        }


def headers(kind):
    response = httpx.get(f"{IDENTITY}/token/{kind}")
    response.raise_for_status()
    return {"Authorization": "Bearer " + response.json()["token"]}


def contract(schema, response):
    spec = yaml.safe_load(SPEC.read_text(encoding="utf-8"))
    Draft202012Validator(
        {"components": spec["components"], "$ref": f"#/components/schemas/{schema}"},
        format_checker=FormatChecker(),
    ).validate(response.json())


def card(response, metric):
    return next(item for item in response.json()["metrics"] if item["metricId"] == metric)


@pytest.mark.integration
@pytest.mark.skipif(
    os.getenv("RUN_ANALYTICS_INTEGRATION") != "1", reason="Requires Compose test stack"
)
def test_runtime(tmp_path):
    settings = Settings.load()
    identity, partitions = kafka.describe(settings)
    probe = kafka.consumer(settings, "test-probe")

    def high():
        return {
            p: probe.get_watermark_offsets(TopicPartition(TOPIC, p), timeout=3)[1]
            for p in partitions
        }

    def cli(*args):
        result = subprocess.run(
            [sys.executable, "-m", "analytics_service", *map(str, args)],
            capture_output=True,
            text=True,
            timeout=15,
        )
        assert result.returncode == 0, result.stderr
        return result.stdout

    bootstrap_manifest = tmp_path / "bootstrap.json"
    bootstrap_manifest.write_bytes(encoded({"topicId": identity, "offsets": high()}))
    cli("bootstrap", bootstrap_manifest)
    producer = Producer(settings.kafka())
    today = datetime.now(TZ).date()
    day = today - timedelta(days=1)
    previous = day - timedelta(days=1)
    boundary = datetime.combine(day, wall_time.min, TZ)
    paid = event("paid", uuid4(), boundary)
    completed = event("completed", uuid4(), boundary + timedelta(seconds=1))
    old_paid = event("paid", uuid4(), boundary - timedelta(seconds=1))
    replay = {**paid, "id": str(uuid4())}

    def publish(payloads):
        for payload in payloads:
            producer.produce(TOPIC, key=payload["data"]["aggregate_id"], value=encoded(payload))
        assert producer.flush(8) == 0
        expected = high()
        wait_for(lambda: current_offsets(settings) == expected)
        return expected

    def certify(day_to_certify, counts, cutoffs):
        manifest = tmp_path / "coverage.json"
        manifest.write_bytes(
            encoded(
                {
                    "topicId": identity,
                    "offsets": cutoffs,
                    "businessDate": str(day_to_certify),
                    "sourceAsOf": datetime.now(UTC).isoformat(),
                    "counts": dict(zip(("PAID", "COMPLETED"), counts, strict=True)),
                    "approvedBy": "integration-test",
                    "evidenceRef": "fixture-oracle",
                }
            )
        )
        approval = cli("coverage", manifest)
        assert cli("coverage", manifest) == approval

    current = publish([paid, completed, old_paid, paid, replay])
    cli("bootstrap", bootstrap_manifest)
    certify(previous, (1, 0), current)
    certify(day, (1, 1), current)
    cli("build-once")
    auth = headers("admin")
    query = {"from": str(day), "to": str(today), "comparison": "previous_period"}
    dashboard = httpx.get(API + "/dashboard", params=query, headers=auth)
    assert dashboard.status_code == 200, dashboard.text
    contract("Dashboard", dashboard)
    assert dashboard.headers["Cache-Control"] == "private, no-store"
    paid_card = card(dashboard, "orders_paid")
    assert paid_card["value"] == "1" and paid_card["quality"] == "FRESH"
    assert paid_card["comparison"]["absoluteChange"] == "0"
    assert card(dashboard, "orders_completed")["value"] == "1"
    assert card(dashboard, "profit")["quality"] == "UNAVAILABLE"
    pinned = dashboard.json()["snapshotId"]
    previous_query = {"from": str(previous), "to": str(day)}
    assert (
        card(
            httpx.get(API + "/dashboard", params=previous_query, headers=auth), "orders_completed"
        )["value"]
        == "0"
    )
    partial = httpx.get(
        API + "/dashboard", params={**query, "to": str(today + timedelta(days=1))}, headers=auth
    )
    assert card(partial, "orders_paid")["quality"] == "PARTIAL"

    stop = Event()
    other = kafka.Collector(settings, stop)
    with db.connect(settings) as conn:
        before = {row["lease_owner"] for row in db.cursors(conn, settings.pipeline)}
    thread = Thread(target=other.run, daemon=True)
    thread.start()
    try:

        def rebalanced():
            with db.connect(settings) as conn:
                owners = {row["lease_owner"] for row in db.cursors(conn, settings.pipeline)}
            return other.owner in owners and bool(owners & (before - {None}))

        wait_for(rebalanced)
        changed = publish([event("paid", uuid4(), boundary + timedelta(seconds=2))])
    finally:
        stop.set()
        thread.join(12)
    snapshots.build_once(settings)
    gap = httpx.get(API + "/dashboard", params=query, headers=auth)
    assert card(gap, "orders_paid")["quality"] == "UNAVAILABLE"
    certify(previous, (1, 0), current)
    certify(day, (2, 1), changed)
    snapshots.build_once(settings)
    updated = httpx.get(API + "/dashboard", params=query, headers=auth)
    assert card(updated, "orders_paid")["value"] == "2"
    assert card(updated, "orders_paid")["comparison"]["absoluteChange"] == "1"
    params = {"from": str(day), "to": str(today), "snapshotId": pinned}
    series = httpx.get(API + "/metrics/orders_paid/series", params=params, headers=auth)
    assert series.status_code == 200, series.text
    contract("MetricSeries", series)
    assert series.json()["buckets"][0]["value"] == "1"

    for kind, status in (("customer", 403), ("warehouse", 403), ("outage", 503)):
        denied = httpx.get(API + "/dashboard", params=query, headers=headers(kind))
        assert denied.status_code == status, denied.text
        contract("Error", denied)
    forged = httpx.get(API + "/dashboard", params=query, headers={"X-User-Role": "ADMIN"})
    assert forged.status_code == 401
    unknown = httpx.get(
        API + "/metrics/orders_paid/series",
        params={**params, "snapshotId": str(uuid4())},
        headers=auth,
    )
    assert unknown.status_code == 404
    malformed = httpx.get(API + "/dashboard", params={**query, "from": str(today)}, headers=auth)
    assert malformed.status_code == 400

    conflict = {**paid, "data": {**paid["data"], "warehouse_id": "other"}}
    final_offsets = publish([conflict])
    snapshots.build_once(settings)
    invalid = httpx.get(API + "/dashboard", params=query, headers=auth)
    assert card(invalid, "orders_paid")["reasonCodes"] == ["SOURCE_INVALID"]
    with db.connect(settings) as conn:
        record = conn.execute(
            "SELECT partition_id,source_offset FROM source_quarantine WHERE pipeline=%s AND resolved_at IS NULL",
            (settings.pipeline,),
        ).fetchone()
        assert record is not None
    cli(
        "repair",
        identity,
        record["partition_id"],
        record["source_offset"],
        "test-discard",
        "--discard",
    )
    certify(day, (2, 1), final_offsets)
    snapshots.build_once(settings)
    assert (
        card(httpx.get(API + "/dashboard", params=query, headers=auth), "orders_paid")["value"]
        == "2"
    )
    client = clickhouse(settings)
    try:
        client.command(
            f"ALTER TABLE {snapshots.DAILY} DELETE WHERE generation_id=toUUID('{pinned}') "
            "SETTINGS mutations_sync=1"
        )
    finally:
        client.close()
    missing_rows = httpx.get(API + "/metrics/orders_paid/series", params=params, headers=auth)
    assert missing_rows.status_code == 503
    contract("Error", missing_rows)
    probe.close()
