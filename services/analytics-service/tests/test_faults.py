import os
from concurrent.futures import ThreadPoolExecutor
from dataclasses import replace
from datetime import datetime, timedelta
from threading import Barrier
from uuid import uuid4

import pytest
from test_analytics import encoded, event

from analytics_service import db, snapshots
from analytics_service.config import Settings
from analytics_service.events import TZ, OrderAdapter
from analytics_service.migrate import clickhouse

pytestmark = [
    pytest.mark.integration,
    pytest.mark.skipif(
        os.getenv("RUN_ANALYTICS_INTEGRATION") != "1", reason="Requires Compose test stack"
    ),
]


@pytest.fixture
def fault_store():
    tag = str(uuid4())
    settings = replace(Settings.load(), pipeline=tag, scope=tag)
    payload = event("paid", uuid4(), datetime.now(TZ) - timedelta(days=1))
    raw = encoded(payload)
    fact = OrderAdapter(settings.schema_dir).decode(payload["data"]["aggregate_id"].encode(), raw)
    with db.connect(settings) as conn:
        db.bootstrap(conn, settings.pipeline, "fault-fixture", {0: 0})
        old = db.claim(conn, settings.pipeline, "fault-fixture", 0, "worker-a")
    return settings, raw, fact, old["lease_epoch"]


def record(conn, settings, raw, fact, epoch, owner="worker-a", offset=0):
    return db.ingest(
        conn,
        settings.pipeline,
        "fault-fixture",
        0,
        offset,
        owner,
        epoch,
        raw,
        fact,
        None,
    )


def test_transaction_rollback_ack_loss_and_lease_fence(fault_store):
    settings, raw, fact, epoch = fault_store

    with pytest.raises(RuntimeError, match="simulated process exit"):
        with db.connect(settings) as conn:
            assert record(conn, settings, raw, fact, epoch) == 1
            raise RuntimeError("simulated process exit")
    with db.connect(settings) as conn:
        assert db.cursors(conn, settings.pipeline)[0]["next_offset"] == 0
        assert (
            conn.execute(
                "SELECT count(*) AS n FROM order_milestone WHERE pipeline=%s",
                (settings.pipeline,),
            ).fetchone()["n"]
            == 0
        )
        assert record(conn, settings, raw, fact, epoch) == 1
    with db.connect(settings) as conn:
        assert record(conn, settings, raw, fact, epoch) == 1
        assert (
            conn.execute(
                "SELECT count(*) AS n FROM order_milestone WHERE pipeline=%s",
                (settings.pipeline,),
            ).fetchone()["n"]
            == 1
        )
        db.release(conn, settings.pipeline, "worker-a", [0])
        current = db.claim(conn, settings.pipeline, "fault-fixture", 0, "worker-b")
    with db.connect(settings) as conn, pytest.raises(db.Conflict, match="lease expired"):
        record(conn, settings, raw, fact, epoch, offset=1)
    with db.connect(settings) as conn:
        assert record(conn, settings, raw, fact, current["lease_epoch"], "worker-b", 2) is None
        cursor = db.cursors(conn, settings.pipeline)[0]
        assert (cursor["next_offset"], cursor["gap_code"]) == (1, "UNEXPECTED_OFFSET")


def test_partial_clickhouse_write_and_competing_publishers(fault_store, monkeypatch):
    settings, raw, fact, epoch = fault_store
    previous = snapshots.build_once(settings)
    with db.connect(settings) as conn:
        assert record(conn, settings, raw, fact, epoch) == 1

    class LoseInsertAck:
        def __init__(self, client):
            self.client = client

        def insert(self, *args, **kwargs):
            self.client.insert(*args, **kwargs)
            raise RuntimeError("simulated loss of ClickHouse insert ACK")

    client = clickhouse(settings)
    try:
        with pytest.raises(RuntimeError, match="ClickHouse insert ACK"):
            snapshots.build(settings, LoseInsertAck(client))
    finally:
        client.close()
    with db.connect(settings) as conn:
        assert (
            db.get_generation(conn, settings.scope, settings.pipeline)["generation_id"] == previous
        )
        assert (
            conn.execute(
                "SELECT count(*) AS n FROM serving_generation WHERE scope_key=%s AND state='ABANDONED'",
                (settings.scope,),
            ).fetchone()["n"]
            == 1
        )

    original = snapshots.materialize
    barrier = Barrier(2, timeout=12)

    def synchronized(*args):
        barrier.wait()
        return original(*args)

    monkeypatch.setattr(snapshots, "materialize", synchronized)
    with ThreadPoolExecutor(max_workers=2) as workers:
        jobs = [workers.submit(snapshots.build_once, settings) for _ in range(2)]
        published = [generation for job in jobs if (generation := job.result())]
    assert len(published) == 1
    with db.connect(settings) as conn:
        assert (
            db.get_generation(conn, settings.scope, settings.pipeline)["generation_id"]
            == published[0]
        )
