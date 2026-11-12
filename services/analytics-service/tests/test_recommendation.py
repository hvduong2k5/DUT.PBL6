import hashlib
import json
import os
from concurrent.futures import ThreadPoolExecutor
from dataclasses import replace
from datetime import UTC, datetime, timedelta
from pathlib import Path
from uuid import UUID, uuid4

import httpx
import psycopg
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient
from psycopg import sql

from analytics_service.events import canonical
from analytics_service.recommendation import engine
from analytics_service.recommendation.api import CustomerAuth, mount
from analytics_service.recommendation.models import Behavior, Config
from analytics_service.recommendation.sources import OWNERS, read_snapshot
from analytics_service.recommendation.store import Error, Store

WAREHOUSE = "a7277eb6-0fd2-4850-b640-169bc446387e"
SCOPE = "d2c-hue"
NOW = datetime.now(UTC)
CUSTOMERS = [str(UUID(int=10000 + n)) for n in range(5)]


def catalog_row(number, category=1, **changes):
    return {
        "recordKey": f"sku:SKU-{number}",
        "skuCode": f"SKU-{number}",
        "productId": str(UUID(int=number)),
        "categoryId": str(UUID(int=category + 100)),
        "productVersion": 1,
        "skuVersion": 1,
        "priceVersion": 1,
        "publiclyVisible": True,
        "skuActive": True,
        "slug": f"product-{number}",
        "name": f"Product {number}",
        "thumbnailUrl": "",
        "priceVnd": 100000,
        "weightGrams": 500,
        "packagingType": "HOP_GIAY_KRAFT",
        "fallbackRank": number,
        "sourceUpdatedAt": NOW.isoformat(),
        **changes,
    }


def inventory_row(number, now=NOW, **changes):
    return {
        "recordKey": f"sku:SKU-{number}",
        "skuCode": f"SKU-{number}",
        "warehouseId": WAREHOUSE,
        "skuActive": True,
        "sellableUnits": 10,
        "observedAt": (now - timedelta(seconds=1)).isoformat(),
        "eligibilityValidUntil": (now + timedelta(seconds=100)).isoformat(),
        **changes,
    }


def basket_row(number, skus, principal=None, now=NOW):
    order = str(UUID(int=100 + number))
    return {
        "recordKey": "order:" + order,
        "orderId": order,
        "orderVersion": 1,
        "cohort": "order-core-1",
        "channel": "D2C_WEB",
        "warehouseId": WAREHOUSE,
        "completedAt": (now - timedelta(seconds=5)).isoformat(),
        "eligible": True,
        "eligibilityReason": None,
        "customerPrincipalId": principal,
        "identityMappingVersion": "identity-core-v1" if principal else None,
        "items": [{"skuCode": sku, "quantityAtCompletion": 1} for sku in sorted(skus)],
    }


def snapshot(config, dataset, records, revision=1, now=NOW):
    snapshot_id = str(uuid4())
    folder = config.export_root / OWNERS[dataset] / snapshot_id
    folder.mkdir(parents=True)
    data = b"".join(canonical(row) + b"\n" for row in sorted(records, key=lambda r: r["recordKey"]))
    at = now - timedelta(seconds=1)
    manifest = {
        "schemaVersion": "1",
        "transport": "FILE_JSONL",
        "dataset": dataset,
        "source": OWNERS[dataset],
        "snapshotId": snapshot_id,
        "sourceRevision": revision,
        "scopeId": SCOPE,
        "channel": "D2C_WEB",
        "warehouseId": WAREHOUSE,
        "sourceAsOf": at.isoformat(),
        "publishedAt": now.isoformat(),
        "expiresAt": (now + timedelta(hours=24)).isoformat(),
        "coverageStatus": "COMPLETE",
        "window": {"from": (at - timedelta(days=90)).isoformat(), "to": at.isoformat()}
        if dataset == "completed_baskets"
        else None,
        "policyVersion": "test-v1",
        "coverageEvidenceRef": "isolated-fixture",
        "recordCount": len(records),
        "pageCount": None,
        "dataFile": "records.jsonl",
        "dataSha256": hashlib.sha256(data).hexdigest(),
    }
    (folder / "records.jsonl").write_bytes(data)
    (folder / "manifest.json").write_bytes(canonical(manifest))
    return folder


def import_rows(store, dataset, records, revision=1, now=NOW):
    manifest, rows = read_snapshot(
        snapshot(store.config, dataset, records, revision, now), store.config, now
    )
    return store.import_snapshot(manifest, rows, now)


@pytest.fixture
def config(tmp_path):
    return Config(
        "unused",
        SCOPE,
        WAREHOUSE,
        "warehouse-supply-pilot",
        b"test-subject-key-32-bytes-long-value",
        export_root=tmp_path,
    )


@pytest.fixture
def store(config):
    if os.getenv("RUN_ANALYTICS_INTEGRATION") != "1":
        pytest.skip("Requires isolated PostgreSQL")
    dsn = os.environ["ANALYTICS_PILOT_PG_DSN"]
    database = "rec_test_" + uuid4().hex
    with psycopg.connect(dsn, autocommit=True) as admin:
        admin.execute(sql.SQL("CREATE DATABASE {}").format(sql.Identifier(database)))
    options = psycopg.conninfo.conninfo_to_dict(dsn)
    options["dbname"] = database
    config = replace(config, dsn=psycopg.conninfo.make_conninfo(**options))
    migration = Path(__file__).resolve().parents[1] / "migrations/postgres/004_recommendation.sql"
    with psycopg.connect(config.dsn) as conn:
        conn.execute(migration.read_text(encoding="utf-8"))
    instance = Store(config)
    import_rows(
        instance, "catalog", [catalog_row(n, category=1 if n < 4 else 2) for n in range(1, 7)]
    )
    import_rows(instance, "sellability", [inventory_row(n) for n in range(1, 7)])
    try:
        yield instance
    finally:
        with psycopg.connect(dsn, autocommit=True) as admin:
            admin.execute(sql.SQL("DROP DATABASE {} WITH (FORCE)").format(sql.Identifier(database)))


def grant(store, principal, now=NOW):
    subject = store.subject(principal)
    store.set_consent(
        subject,
        {"enabled": True, "policyVersion": "personalization-v1"},
        0,
        uuid4(),
        now - timedelta(days=1),
    )
    return subject


def test_content_scores_and_product_dedup():
    anchor = catalog_row(1)
    assert engine.similarity(anchor, catalog_row(2, weightGrams=450, priceVnd=95000)) == 4
    assert engine.similarity(anchor, catalog_row(3, weightGrams=250, priceVnd=65000)) == 2
    candidate = catalog_row(2)
    same_product = catalog_row(3, productId=candidate["productId"])
    selected = engine.select(
        [("SKU-2", "a", "r"), ("SKU-3", "a", "r")],
        {"SKU-2": candidate, "SKU-3": same_product},
        set(),
        6,
    )
    assert len(selected) == 1 and selected[0]["skuCode"] == "SKU-2"


def test_pair_computation_has_an_explicit_budget(monkeypatch):
    monkeypatch.setattr(engine, "MAX_PAIR_UPDATES", 1)
    with pytest.raises(engine.BudgetExceeded):
        engine.pairs([basket_row(1, ["SKU-1", "SKU-2", "SKU-3"])], {})
    with pytest.raises(engine.BudgetExceeded):
        engine.collaborative({"subject": {"SKU-1": 1, "SKU-2": 1, "SKU-3": 1}})


def test_identity_mapping_version_is_owner_approved(config):
    row = basket_row(1, ["SKU-1"], CUSTOMERS[0])
    row["identityMappingVersion"] = "unapproved-version"
    with pytest.raises(Error, match="INVALID_BASKET"):
        read_snapshot(snapshot(config, "completed_baskets", [row]), config, NOW)


def test_pairs_support_lift_and_single_item_denominator():
    catalog = {f"SKU-{n}": catalog_row(n) for n in range(1, 5)}
    baskets = [basket_row(n, ["SKU-1", "SKU-2"]) for n in range(5)]
    baskets += [basket_row(n + 5, ["SKU-3"]) for n in range(15)]
    result = engine.pairs(baskets, catalog)
    assert result["total"] == 20
    assert result["neighbors"]["SKU-1"] == [{"sku": "SKU-2", "common": 5, "anchorCount": 5}]
    baskets[0]["eligible"] = False
    assert engine.pairs(baskets, catalog)["neighbors"] == {}


def test_history_caps_decay_and_privacy_boundary():
    consent = {"enabled": True, "version": 1, "granted_at": NOW - timedelta(days=10)}
    base = {
        "subject_key": "u",
        "sku_code": "SKU-1",
        "event_type": "VIEW_ITEM",
        "effective_at": NOW - timedelta(days=7),
        "consent_version": 1,
    }
    profiles = engine.history([base, base], [], {"u": consent}, NOW)
    assert profiles["u"]["SKU-1"] == pytest.approx(0.5)
    assert engine.history([base], [], {"u": {**consent, "enabled": False}}, NOW) == {}
    assert engine.collaborative({"a": {"A": 1, "B": 1}, "b": {"A": 1, "B": 1}}) == {}
    rows = engine.collaborative({str(n): {"A": 1, "B": 1} for n in range(3)})
    assert rows["A"][0]["score"] == pytest.approx(3 / 13)


@pytest.mark.parametrize(
    "change,code",
    [
        ({"scopeId": "wrong"}, "SOURCE_SCOPE_MISMATCH"),
        ({"recordCount": 2}, "EXPORT_CHECKSUM_MISMATCH"),
        ({"dataSha256": "0" * 64}, "EXPORT_CHECKSUM_MISMATCH"),
        ({"sourceAsOf": (NOW + timedelta(minutes=1)).isoformat()}, "INVALID_SOURCE_TIME"),
        ({"unexpected": "value"}, "INVALID_MANIFEST"),
    ],
)
def test_intake_rejects_bad_manifest(config, change, code):
    folder = snapshot(config, "catalog", [catalog_row(1)])
    path = folder / "manifest.json"
    path.write_bytes(canonical({**json.loads(path.read_bytes()), **change}))
    with pytest.raises(Error, match=code):
        read_snapshot(folder, config, NOW)


def test_intake_checks_record_shape_order_and_expiry(config):
    folder = snapshot(config, "sellability", [inventory_row(1, sellableUnits=-1)])
    with pytest.raises(Error, match="INVALID_SOURCE_RECORD"):
        read_snapshot(folder, config, NOW)
    folder = snapshot(config, "catalog", [catalog_row(1), catalog_row(1)])
    with pytest.raises(Error, match="INVALID_RECORD_ORDER"):
        read_snapshot(folder, config, NOW)
    folder = snapshot(config, "sellability", [inventory_row(1, warehouseId=str(uuid4()))])
    with pytest.raises(Error, match="INVALID_SELLABILITY"):
        read_snapshot(folder, config, NOW)


def test_complete_empty_snapshot(config):
    manifest, records = read_snapshot(snapshot(config, "catalog", []), config, NOW)
    assert manifest.recordCount == 0 and records == []


def test_store_duplicate_conflict_and_stale_revision(store):
    folder = snapshot(store.config, "catalog", [catalog_row(1)], revision=7)
    manifest, records = read_snapshot(folder, store.config, NOW)
    assert store.import_snapshot(manifest, records, NOW) == "IMPORTED"
    assert store.import_snapshot(manifest, records, NOW) == "DUPLICATE"
    with pytest.raises(Error, match="SNAPSHOT_CONFLICT"):
        store.import_snapshot(
            manifest.model_copy(update={"policyVersion": "changed"}), records, NOW
        )
    with pytest.raises(Error, match="SNAPSHOT_CONFLICT"):
        import_rows(store, "catalog", [catalog_row(2)], revision=7)
    with pytest.raises(Error, match="STALE_SOURCE_REVISION"):
        import_rows(store, "catalog", [], revision=6)


def test_full_replacement_and_freshness(store):
    result = store.recommend(
        "product_detail", {"productId": str(UUID(int=1)), "skuCode": "SKU-1"}, 6, None, NOW
    )
    assert result["strategy"] == "RELATED"
    assert result["items"][0]["skuCode"] == "SKU-2"
    import_rows(store, "sellability", [inventory_row(3)], 2)
    result = store.recommend("product_detail", {"productId": str(UUID(int=1))}, 6, None, NOW)
    assert [r["skuCode"] for r in result["items"]] == ["SKU-3"]
    with pytest.raises(Error, match="SOURCE_UNAVAILABLE"):
        store.recommend("home", {}, 6, None, NOW + timedelta(seconds=121))


def test_basket_recommendations_and_correction(store):
    baskets = [basket_row(n, ["SKU-1", "SKU-4"]) for n in range(5)]
    baskets += [basket_row(n + 5, ["SKU-2"]) for n in range(15)]
    import_rows(store, "completed_baskets", baskets)
    generation = store.build(NOW)
    assert store.build(NOW) == generation
    result = store.recommend("cart", {"skuCodes": ["SKU-1"]}, 6, None, NOW)
    assert result["strategy"] == "BOUGHT_TOGETHER" and result["items"][0]["skuCode"] == "SKU-4"
    assert all(r["skuCode"] != "SKU-1" for r in result["items"])
    import_rows(store, "completed_baskets", baskets[5:], 2)
    pending = store.recommend("cart", {"skuCodes": ["SKU-1"]}, 6, None, NOW)
    assert pending["strategy"] == "FALLBACK" and "SOURCE_CHANGED_REBUILD" in pending["reasonCodes"]
    store.build(NOW)
    result = store.recommend("cart", {"skuCodes": ["SKU-1"]}, 6, None, NOW)
    assert result["strategy"] == "FALLBACK" and "INSUFFICIENT_BASKETS" in result["reasonCodes"]


def test_purchase_mapping_personalization_and_revoke(store):
    subjects = [grant(store, principal) for principal in CUSTOMERS]
    baskets = [basket_row(n, ["SKU-4", "SKU-5"], CUSTOMERS[n]) for n in range(3)]
    baskets.append(basket_row(5, ["SKU-2"], CUSTOMERS[3]))
    import_rows(store, "completed_baskets", baskets)
    store.build(NOW)
    a = store.recommend("home", {}, 6, subjects[0], NOW)
    b = store.recommend("home", {}, 6, subjects[3], NOW)
    assert a["items"][0]["algorithmVersion"] == "item_cosine_v1"
    assert a["items"][0]["skuCode"] in {"SKU-4", "SKU-5"}
    assert b["items"][0]["skuCode"] in {"SKU-1", "SKU-2", "SKU-3"}
    with store.connect() as conn:
        raw = json.dumps(store.sources(conn)["completed_baskets"]["records"])
    assert all(principal not in raw for principal in CUSTOMERS)
    store.set_consent(subjects[0], {"enabled": False}, 1, uuid4(), NOW)
    after = store.recommend("home", {}, 6, subjects[0], NOW)
    assert after["strategy"] == "FALLBACK"
    store.cleanup(NOW)
    with store.connect() as conn:
        records = store.sources(conn)["completed_baskets"]["records"]
    assert not any(r.get("subjectKey") == subjects[0] for r in records)
    assert any(r.get("subjectKey") == subjects[1] for r in records)


def test_event_idempotency_cap_and_consent_conflict(store):
    actor = grant(store, CUSTOMERS[0])
    event = Behavior.model_validate_json(
        json.dumps(
            {
                "eventId": str(uuid4()),
                "eventType": "VIEW_ITEM",
                "skuCode": "SKU-4",
                "occurredAt": NOW.isoformat(),
                "consentVersion": 1,
            }
        )
    )
    assert store.accept_event(actor, event, NOW) == "ACCEPTED"
    assert store.accept_event(actor, event, NOW + timedelta(seconds=1)) == "DUPLICATE"
    with pytest.raises(Error, match="EVENT_ID_CONFLICT"):
        store.accept_event(actor, event.model_copy(update={"skuCode": "SKU-3"}), NOW)
    store.set_consent(actor, {"enabled": False}, 1, uuid4(), NOW)
    with pytest.raises(Error, match="CONSENT_VERSION_CONFLICT"):
        store.accept_event(actor, event, NOW)


def test_consent_concurrent_writes_and_idempotency(store):
    actor = store.subject(CUSTOMERS[0])

    def enable():
        try:
            return store.set_consent(actor, {"enabled": True}, 0, uuid4(), NOW)["consentVersion"]
        except Error as exc:
            return exc.status

    with ThreadPoolExecutor(max_workers=2) as pool:
        assert sorted(pool.map(lambda _: enable(), range(2))) == [1, 412]
    key = uuid4()
    assert store.set_consent(actor, {"enabled": False}, 1, key, NOW)["consentVersion"] == 2
    assert store.set_consent(actor, {"enabled": False}, 1, key, NOW)["consentVersion"] == 2
    with pytest.raises(Error, match="IDEMPOTENCY_CONFLICT"):
        store.set_consent(actor, {"enabled": True}, 2, key, NOW)


def test_revoke_during_build_fences_generation(store, monkeypatch):
    store.config = replace(store.config, inventory_age=1000)
    actor = grant(store, CUSTOMERS[0])
    initial = store.build(NOW)
    original = engine.collaborative

    def revoke(profiles):
        store.set_consent(actor, {"enabled": False}, 1, uuid4(), NOW)
        return original(profiles)

    monkeypatch.setattr(engine, "collaborative", revoke)
    with pytest.raises(Error, match="BUILD_INPUT_CHANGED"):
        store.build(NOW + timedelta(seconds=300))
    with store.connect() as conn:
        assert (
            str(
                conn.execute("SELECT generation_id FROM serving_pointer").fetchone()[
                    "generation_id"
                ]
            )
            == initial
        )


def test_erasure_preserves_history_after_regrant(store):
    actor = grant(store, CUSTOMERS[0])

    def event(at, version):
        return Behavior.model_validate_json(
            json.dumps(
                {
                    "eventId": str(uuid4()),
                    "eventType": "VIEW_ITEM",
                    "skuCode": "SKU-1",
                    "occurredAt": at.isoformat(),
                    "consentVersion": version,
                }
            )
        )

    store.accept_event(actor, event(NOW, 1), NOW)
    store.set_consent(actor, {"enabled": False}, 1, uuid4(), NOW)
    store.set_consent(actor, {"enabled": True}, 2, uuid4(), NOW + timedelta(seconds=1))
    store.accept_event(actor, event(NOW + timedelta(seconds=2), 3), NOW + timedelta(seconds=2))
    store.cleanup(NOW + timedelta(seconds=3))
    with store.connect() as conn:
        rows = conn.execute("SELECT consent_version FROM behavior_event").fetchall()
    assert rows == [{"consent_version": 3}]


def test_source_unknown_fields_cannot_import_private_data(config):
    row = catalog_row(1, customerEmail="private@example.test")
    with pytest.raises(Error, match="INVALID_SOURCE_RECORD"):
        read_snapshot(snapshot(config, "catalog", [row]), config, NOW)


def test_concurrent_builds_have_one_published_generation(store):
    with ThreadPoolExecutor(max_workers=2) as pool:
        ids = list(pool.map(lambda _: store.build(NOW), range(2)))
    assert ids[0] == ids[1]
    with store.connect() as conn:
        assert conn.execute("SELECT count(*) AS n FROM generation").fetchone()["n"] == 1


class TestIdentity:
    __test__ = False

    def authenticate(self, request):
        token = request.headers.get("authorization", "").removeprefix("Bearer ")
        if token not in CUSTOMERS:
            raise Error(401, "UNAUTHENTICATED")
        return token


def test_actual_api_validation_and_consent_flow(store):
    now = datetime.now(UTC)
    import_rows(store, "catalog", [catalog_row(n) for n in range(1, 7)], 2, now)
    import_rows(store, "sellability", [inventory_row(n, now) for n in range(1, 7)], 2, now)
    app = FastAPI()
    mount(app, store.config, TestIdentity())
    with TestClient(app) as client:
        assert client.get("/api/v1/products/recommended?placement=home&userId=x").status_code == 400
        assert client.get("/api/v1/products/recommended?placement=home&limit=0").status_code == 400
        assert (
            client.get("/api/v1/products/recommended?placement=home&placement=cart").status_code
            == 400
        )
        assert (
            client.get("/api/v1/products/recommended?placement=cart").json()["strategy"] == "EMPTY"
        )
        headers = {"Authorization": "Bearer " + CUSTOMERS[0]}
        consent = "/api/v1/recommendations/consent"
        assert client.get(consent).status_code == 401
        assert client.get(consent, headers=headers).headers["etag"] == '"0"'
        body = {"enabled": True, "policyVersion": "personalization-v1"}
        assert client.put(consent, json=body, headers=headers).status_code == 428
        headers.update({"If-Match": '"0"', "Idempotency-Key": str(uuid4())})
        assert client.put(consent, json=body, headers=headers).status_code == 200
        event = {
            "eventId": str(uuid4()),
            "eventType": "VIEW_ITEM",
            "skuCode": "SKU-4",
            "occurredAt": datetime.now(UTC).isoformat(),
            "consentVersion": 1,
        }
        endpoint = "/api/v1/recommendations/events"
        assert client.post(endpoint, json=event, headers=headers).status_code == 202
        assert client.post(endpoint, json=event, headers=headers).status_code == 200
        assert (
            client.post(
                endpoint, json={**event, "eventType": "PURCHASE"}, headers=headers
            ).status_code
            == 400
        )
        assert (
            client.post(
                endpoint,
                content=b"x" * 4097,
                headers={**headers, "Content-Type": "application/json"},
            ).status_code
            == 413
        )


def test_customer_auth_header_serialization_with_mock_transport():
    from starlette.requests import Request

    from analytics_service.config import AuthSettings

    settings = AuthSettings(
        "issuer", "unused", "http://identity.test", frozenset(), session_origin="http://web.test"
    )
    auth = CustomerAuth(settings)
    observed = []

    def handler(request):
        observed.append(request.headers.get("cookie"))
        return httpx.Response(
            200,
            json={
                "success": True,
                "data": {
                    "principalId": CUSTOMERS[0],
                    "principalType": "CUSTOMER",
                    "provisioningStatus": "ACTIVE",
                },
            },
        )

    auth.client = httpx.Client(transport=httpx.MockTransport(handler))
    request = Request(
        {
            "type": "http",
            "method": "GET",
            "headers": [(b"cookie", b"OMAMA_SESSION=f47ac10b-58cc-4372-a567-0e02b2c3d477")],
        }
    )
    assert auth.authenticate(request) == CUSTOMERS[0]
    assert observed == ["OMAMA_SESSION=f47ac10b-58cc-4372-a567-0e02b2c3d477"]
