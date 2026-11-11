import logging
from datetime import UTC, datetime
from pathlib import Path

from prometheus_client import Counter, Gauge

from . import db, supply
from .config import positive, setting
from .events import digest

log = logging.getLogger(__name__)
FILE_LIMIT = 1_048_576
RESULT = Counter("analytics_supply_files", "Supply intake results", ["source", "result"])
SEEN = Gauge(
    "analytics_supply_source_as_of_seconds", "Latest source timestamp seen", ["source", "kind"]
)
AVAILABLE = Gauge("analytics_supply_inbox_available", "Owner inbox available", ["source"])
KINDS = {
    "order-service": ("order_day",),
    "inventory-service": ("inventory_day", "inventory_batches"),
}


class Inbox:
    def __init__(self, settings, root=None):
        self.settings = settings
        self.root = Path(root or setting("SUPPLY_INBOX", "/run/supply"))
        self.interval = positive("SUPPLY_SYNC_INTERVAL_SECONDS", 60)
        self.processed = {}
        self.latest = {}
        for owner, kinds in KINDS.items():
            for kind in kinds:
                SEEN.labels(owner, kind).set(0)

    def run_once(self):
        current = {}
        for owner in KINDS:
            folder = self.root / owner
            try:
                paths = (
                    sorted(path for path in folder.iterdir() if path.suffix == ".json")
                    if not folder.is_symlink()
                    else None
                )
            except OSError:
                paths = None
            AVAILABLE.labels(owner).set(paths is not None)
            if paths is None:
                continue
            for path in paths:
                if path.is_symlink() or not path.is_file():
                    continue
                try:
                    with path.open("rb") as handle:
                        body = handle.read(FILE_LIMIT + 1)
                except OSError:
                    RESULT.labels(owner, "read_error").inc()
                    continue
                fingerprint = digest(body)
                if self.processed.get(path) == fingerprint:
                    current[path] = fingerprint
                    continue
                try:
                    if len(body) > FILE_LIMIT:
                        raise db.Conflict("Supply file exceeds size limit")
                    export = supply.parse_export(body, datetime.now(UTC))
                    if export.source != owner:
                        raise db.Conflict("Supply file arrived in a different owner's inbox")
                    with db.connect(self.settings) as conn:
                        result = supply.import_export(conn, self.settings, export)
                    stamp = export.sourceAsOf.timestamp()
                    key = (owner, export.kind)
                    self.latest[key] = max(self.latest.get(key, 0), stamp)
                    SEEN.labels(*key).set(self.latest[key])
                    current[path] = fingerprint
                    RESULT.labels(owner, result).inc()
                except supply.SourceNotYetCurrent:
                    RESULT.labels(owner, "retry").inc()
                except db.Conflict:
                    current[path] = fingerprint
                    RESULT.labels(owner, "rejected").inc()
                    log.warning(
                        "Supply export rejected source=%s digest=%s", owner, fingerprint.hex()
                    )
                except Exception as exc:
                    RESULT.labels(owner, "retry").inc()
                    log.warning(
                        "Supply import will retry source=%s error=%s", owner, type(exc).__name__
                    )
        self.processed = current
