import json
import logging
import signal
from datetime import UTC, datetime
from threading import Event

from prometheus_client import start_http_server

from .models import Config
from .sources import sync_once
from .store import Error, Store

log = logging.getLogger(__name__)


def run(command, generation=None):
    store = Store(Config.load())
    if command == "recommendation-build-once":
        print(json.dumps({"generationId": store.build(datetime.now(UTC))}))
    elif command == "recommendation-status":
        with store.connect() as conn:
            sources = store.sources(conn)
            pointer = conn.execute(
                "SELECT generation_id FROM serving_pointer WHERE scope_id=%s", (store.config.scope,)
            ).fetchone()
            print(
                json.dumps(
                    {
                        "scopeId": store.config.scope,
                        "generationId": str(pointer["generation_id"]) if pointer else None,
                        "sources": {
                            name: {
                                "revision": r["source_revision"],
                                "asOf": r["source_as_of"].isoformat(),
                                "recordCount": r["manifest"]["recordCount"],
                            }
                            for name, r in sources.items()
                        },
                    },
                    indent=2,
                )
            )
    elif command == "recommendation-cleanup":
        store.cleanup(datetime.now(UTC))
    elif command == "recommendation-rollback":
        with store.connect() as conn:
            epoch = store.lock_privacy(conn)
            row = conn.execute(
                """SELECT * FROM generation WHERE generation_id=%s AND scope_id=%s
                   AND built_at > clock_timestamp()-interval '24 hours'""",
                (generation, store.config.scope),
            ).fetchone()
            if not row or row["privacy_epoch"] != epoch:
                raise Error(409, "GENERATION_NOT_ELIGIBLE")
            store.check_sources(store.sources(conn), datetime.now(UTC))
            conn.execute(
                "UPDATE serving_pointer SET generation_id=%s WHERE scope_id=%s",
                (generation, store.config.scope),
            )
    else:
        start_http_server(8015)
        stop = Event()
        for name in (signal.SIGINT, signal.SIGTERM):
            signal.signal(name, lambda *_: stop.set())
        while not stop.is_set():
            try:
                store.cleanup(datetime.now(UTC))
                sync_once(store)
                store.build(datetime.now(UTC))
            except Exception as exc:
                log.warning("Recommendation cycle failed type=%s", type(exc).__name__)
            stop.wait(store.config.interval)
