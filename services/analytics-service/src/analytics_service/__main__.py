import argparse
import json
import logging
import signal
from datetime import UTC, date, datetime
from pathlib import Path
from threading import Event
from uuid import UUID

import uvicorn
from prometheus_client import start_http_server

from . import db, kafka, snapshots, supply, supply_sync
from .config import Settings, setting
from .events import OrderAdapter
from .migrate import migrate

log = logging.getLogger(__name__)


def read_manifest(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))


def offsets(manifest):
    positions = manifest["offsets"]
    if not positions or any(
        not key.isdecimal() or str(int(key)) != key or type(value) is not int or value < 0
        for key, value in positions.items()
    ):
        raise db.Conflict("Offsets require decimal partition keys and nonnegative integers")
    return {int(partition): offset for partition, offset in positions.items()}


def worker(settings, name):
    stop = Event()
    for signum in (signal.SIGINT, signal.SIGTERM):
        signal.signal(signum, lambda *_: stop.set())
    start_http_server({"collector": 8012, "builder": 8013, "supply-sync": 8014}[name])
    sync = supply_sync.Inbox(settings) if name == "supply-sync" else None
    while not stop.is_set():
        try:
            if name == "collector":
                kafka.Collector(settings, stop).run()
            elif sync:
                sync.run_once()
                stop.wait(sync.interval)
            else:
                generation = snapshots.build_once(settings)
                if generation:
                    log.info("Published generation %s", generation)
                stop.wait(settings.build_interval)
        except Exception as exc:
            log.warning("%s cycle failed: %s", name, type(exc).__name__)
            stop.wait(3)


def parse_args():
    parser = argparse.ArgumentParser()
    commands = parser.add_subparsers(dest="command", required=True)
    for command in (
        "api",
        "collector",
        "builder",
        "supply-sync",
        "migrate",
        "status",
        "topic",
        "build-once",
        "recommendation-sync",
        "recommendation-build-once",
        "recommendation-status",
        "recommendation-cleanup",
    ):
        commands.add_parser(command)
    for command in ("bootstrap", "coverage", "import-supply"):
        commands.add_parser(command).add_argument("manifest")
    commands.add_parser("rollback").add_argument("generation", type=UUID)
    commands.add_parser("recommendation-rollback").add_argument("generation", type=UUID)
    repair = commands.add_parser("repair")
    repair.add_argument("topic_id")
    repair.add_argument("partition", type=int)
    repair.add_argument("offset", type=int)
    repair.add_argument("evidence")
    repair.add_argument(
        "--discard", action="store_true", help="Owner approved discarding the record"
    )
    return parser.parse_args()


def main():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
    args = parse_args()
    settings = Settings.load()
    command = args.command
    if command.startswith("recommendation-"):
        from .recommendation.jobs import run

        run(command, getattr(args, "generation", None))
        return
    if command == "api":
        uvicorn.run(
            "analytics_service.api:create_app",
            factory=True,
            host="0.0.0.0",
            port=int(setting("PORT", "8011")),
            access_log=False,
        )
    elif command in ("collector", "builder", "supply-sync"):
        worker(settings, command)
    elif command == "migrate":
        migrate(settings)
    elif command == "topic":
        identity, partitions = kafka.describe(settings)
        print(json.dumps({"topicId": identity, "partitions": sorted(partitions)}))
    elif command == "status":
        with db.connect(settings) as conn:
            rows = db.cursors(conn, settings.pipeline)
            keys = ("partition_id", "topic_identity", "next_offset", "gap_code")
            print(
                json.dumps(
                    {
                        "pipeline": settings.pipeline,
                        "cursors": [{key: row[key] for key in keys} for row in rows],
                        "openQuarantine": db.open_quarantine(conn, settings.pipeline),
                    }
                )
            )
    elif command == "bootstrap":
        manifest = read_manifest(args.manifest)
        kafka.bootstrap(settings, manifest["topicId"], offsets(manifest))
    elif command == "coverage":
        manifest = read_manifest(args.manifest)
        identity, partitions = kafka.describe(settings)
        proof_offsets = offsets(manifest)
        if manifest["topicId"] != identity or set(proof_offsets) != partitions:
            raise db.Conflict("Coverage does not match broker metadata")
        with db.connect(settings) as conn:
            approval_id = db.approve_day(
                conn,
                settings.pipeline,
                date.fromisoformat(manifest["businessDate"]),
                identity,
                proof_offsets,
                manifest["counts"]["PAID"],
                manifest["counts"]["COMPLETED"],
                datetime.fromisoformat(manifest["sourceAsOf"]),
                manifest["approvedBy"],
                manifest["evidenceRef"],
            )
        print(json.dumps({"approvalId": approval_id}))
    elif command == "build-once":
        print(snapshots.build_once(settings))
    elif command == "import-supply":
        with Path(args.manifest).open("rb") as manifest:
            body = manifest.read(1_048_577)
        if len(body) > 1_048_576:
            raise db.Conflict("Supply export exceeds 1 MiB")
        export = supply.parse_export(body, datetime.now(UTC))
        with db.connect(settings) as conn:
            result = supply.import_export(conn, settings, export)
        print(json.dumps({"status": result, "kind": export.kind, "revision": export.revision}))
    elif command == "rollback":
        snapshots.rollback(settings, args.generation)
    elif command == "repair":
        if not args.evidence.strip():
            raise db.Conflict("Repair requires an owner evidence reference")
        raw = event = None
        if not args.discard:
            identity, _ = kafka.describe(settings)
            if identity != args.topic_id:
                raise db.Conflict("Kafka topic identity has changed")
            probe = kafka.consumer(settings, f"{settings.pipeline}-repair")
            try:
                message = kafka.read_exact(probe, args.partition, args.offset)
                raw = message.value() or b""
                event = OrderAdapter(settings.schema_dir).decode(message.key(), raw)
            finally:
                probe.close()
        with db.connect(settings) as conn:
            db.repair(
                conn,
                settings.pipeline,
                args.topic_id,
                args.partition,
                args.offset,
                args.evidence,
                raw,
                event,
            )


if __name__ == "__main__":
    main()
