"""Run beside an owner DB with read-only libpq credentials; publish complete JSONL snapshots."""

import argparse
import hashlib
import json
import os
import re
import shutil
import signal
import subprocess
import sys
import tempfile
from contextlib import contextmanager
from datetime import UTC, datetime, timedelta
from pathlib import Path
from threading import Event
from uuid import UUID, uuid4

ROOT = Path(__file__).resolve().parents[1]
OWNERS = {
    "catalog": "catalog-service",
    "sellability": "inventory-service",
    "completed_baskets": "order-service",
}
QUERIES = {
    "catalog": "analytics-recommendation-catalog.sql",
    "sellability": "analytics-recommendation-sellability.sql",
    "completed_baskets": "analytics-recommendation-baskets.sql",
}


def canonical(value):
    return json.dumps(value, sort_keys=True, ensure_ascii=False, separators=(",", ":")).encode()


@contextmanager
def publisher_lock(path):
    with path.open("a+b") as handle:
        if os.name == "nt":
            import msvcrt

            if handle.tell() == 0:
                handle.write(b"0")
                handle.flush()
            handle.seek(0)
            msvcrt.locking(handle.fileno(), msvcrt.LK_NBLCK, 1)
        else:
            import fcntl

            fcntl.flock(handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
        try:
            yield
        finally:
            if os.name == "nt":
                handle.seek(0)
                msvcrt.locking(handle.fileno(), msvcrt.LK_UNLCK, 1)


def export(dataset, scope, target):
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,63}", scope["scopeId"]):
        raise ValueError("Invalid scope")
    warehouse = str(UUID(scope["warehouseId"]))
    if scope["channel"] != "D2C_WEB" or scope["orderCohort"] != "order-core-1":
        raise ValueError("Only approved D2C/core cohorts can be exported")
    mapping = scope.get("identityMappingVersion", "")
    if dataset == "completed_baskets" and not re.fullmatch(r"[A-Za-z0-9_.-]{1,128}", mapping):
        raise ValueError("Owner-approved identityMappingVersion required for purchase history")
    excluded = [str(UUID(value)) for value in scope.get("excludedOrderIds", [])]
    owner = OWNERS[dataset]
    folder = target.resolve() / owner
    folder.mkdir(parents=True, exist_ok=True)
    if folder.is_symlink():
        raise ValueError("Owner export folder must not be a symlink")
    with publisher_lock(folder / ("." + scope["scopeId"] + ".lock")):
        query = ROOT / "services" / owner / "scripts" / QUERIES[dataset]
        result = subprocess.run(
            [
                "psql",
                "-X",
                "-q",
                "-t",
                "-A",
                "-v",
                "ON_ERROR_STOP=1",
                "-v",
                f"warehouse={warehouse}",
                "-v",
                f"identity_mapping={mapping}",
                "-v",
                "excluded_orders=" + ",".join(excluded),
            ],
            input="BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;\nSET LOCAL TIME ZONE 'UTC';\n"
            + query.read_text()
            + "\nCOMMIT;\n",
            capture_output=True,
            check=False,
            text=True,
            timeout=30,
            encoding="utf-8",
        )
        if result.returncode:
            raise ValueError("Owner export query failed; inspect owner DB logs")
        captured = json.loads(result.stdout)
        if captured["violations"]:
            raise ValueError("Owner quantity reconciliation failed")
        records = sorted(captured["records"], key=lambda r: r["recordKey"])
        if len(records) > (100_000 if dataset == "completed_baskets" else 1000):
            raise ValueError("Owner export record budget exceeded")
        data = b"".join(canonical(record) + b"\n" for record in records)
        if len(data) > 64 * 1024 * 1024:
            raise ValueError("Owner export file exceeds 64 MiB")
        counter = folder / ("." + scope["scopeId"] + ".revision")
        revision = int(counter.read_text()) + 1 if counter.exists() else 1
        previous = []
        for existing in folder.glob("*/manifest.json"):
            manifest = json.loads(existing.read_text(encoding="utf-8"))
            if manifest["scopeId"] == scope["scopeId"]:
                previous.append(manifest["sourceRevision"])
        if previous and revision <= max(previous):
            raise ValueError("Owner revision counter requires reconciliation")
        snapshot = str(uuid4())
        now = datetime.now(UTC)
        manifest = {
            "schemaVersion": "1",
            "transport": "FILE_JSONL",
            "dataset": dataset,
            "source": owner,
            "snapshotId": snapshot,
            "sourceRevision": revision,
            "scopeId": scope["scopeId"],
            "channel": scope["channel"],
            "warehouseId": warehouse,
            "sourceAsOf": captured["sourceAsOf"],
            "publishedAt": now.isoformat(),
            "expiresAt": (now + timedelta(days=1)).isoformat(),
            "window": captured["window"],
            "coverageStatus": "COMPLETE",
            "policyVersion": "recommendation-export-v1",
            "coverageEvidenceRef": f"owner-query/{scope['scopeId']}/{snapshot}",
            "recordCount": len(records),
            "pageCount": None,
            "dataFile": "records.jsonl",
            "dataSha256": hashlib.sha256(data).hexdigest(),
        }
        staging = Path(tempfile.mkdtemp(prefix=".staging-", dir=folder))
        try:
            for name, body in (
                ("records.jsonl", data),
                ("manifest.json", canonical(manifest)),
            ):
                with (staging / name).open("wb") as handle:
                    handle.write(body)
                    handle.flush()
                    os.fsync(handle.fileno())
                os.chmod(staging / name, 0o640)
            counter_temp = folder / ("." + snapshot + ".counter")
            with counter_temp.open("w") as handle:
                handle.write(str(revision))
                handle.flush()
                os.fsync(handle.fileno())
            os.replace(counter_temp, counter)
            staging.rename(folder / snapshot)
            if os.name != "nt":
                descriptor = os.open(folder, os.O_DIRECTORY)
                try:
                    os.fsync(descriptor)
                finally:
                    os.close(descriptor)
        finally:
            if staging.exists() and staging.parent.resolve() == folder.resolve():
                shutil.rmtree(staging)
        return {
            "snapshotId": snapshot,
            "recordCount": len(records),
            "revision": revision,
        }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("dataset", choices=OWNERS)
    parser.add_argument("scope", type=Path)
    parser.add_argument("outbox", type=Path)
    parser.add_argument("--watch", action="store_true", help="Publish repeatedly until stopped")
    parser.add_argument("--interval", type=int, help="Seconds between exports (default by dataset)")
    args = parser.parse_args()
    interval = (
        args.interval
        if args.interval is not None
        else {"catalog": 300, "sellability": 60, "completed_baskets": 900}[args.dataset]
    )
    if interval < 1:
        parser.error("interval must be positive")
    scope = json.loads(args.scope.read_text(encoding="utf-8"))
    stop = Event()
    if args.watch:
        for signum in (signal.SIGINT, signal.SIGTERM):
            signal.signal(signum, lambda *_: stop.set())
    while not stop.is_set():
        try:
            print(json.dumps(export(args.dataset, scope, args.outbox)), flush=True)
        except Exception as exc:
            if not args.watch:
                raise
            print(f"Export failed type={type(exc).__name__}", file=sys.stderr, flush=True)
        if not args.watch:
            break
        stop.wait(interval)


if __name__ == "__main__":
    main()
