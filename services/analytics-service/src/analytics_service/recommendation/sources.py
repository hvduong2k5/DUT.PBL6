import hashlib
import json
import logging
from datetime import UTC, datetime, timedelta
from pathlib import Path

from prometheus_client import Counter, Gauge
from pydantic import ValidationError

from ..events import canonical
from .models import Basket, Catalog, Manifest, Sellability
from .store import Error

log = logging.getLogger(__name__)
OWNERS = {
    "catalog": "catalog-service",
    "sellability": "inventory-service",
    "completed_baskets": "order-service",
}
MODELS = {"catalog": Catalog, "sellability": Sellability, "completed_baskets": Basket}
FILE_LIMIT = 64 * 1024 * 1024
SOURCE_AS_OF = Gauge(
    "analytics_recommendation_source_as_of_seconds", "Current source cutoff", ["source", "scope"]
)
IMPORTS = Counter(
    "analytics_recommendation_imports", "Source intake outcomes", ["source", "result"]
)


def read_snapshot(folder, config, now):
    root = config.export_root.resolve()
    folder = Path(folder)
    if folder.is_symlink() or folder.parent.is_symlink() or root not in folder.resolve().parents:
        raise Error(400, "UNSAFE_EXPORT_PATH")
    manifest_file = folder / "manifest.json"
    data_file = folder / "records.jsonl"
    if manifest_file.is_symlink() or data_file.is_symlink():
        raise Error(400, "UNSAFE_EXPORT_PATH")
    with manifest_file.open("rb") as handle:
        raw = handle.read(16385)
    if len(raw) > 16384:
        raise Error(400, "MANIFEST_TOO_LARGE")
    try:
        manifest = Manifest.model_validate_json(raw)
    except ValidationError as exc:
        raise Error(400, "INVALID_MANIFEST") from exc
    if (
        manifest.scopeId != config.scope
        or str(manifest.warehouseId) != config.warehouse
        or manifest.source != OWNERS[manifest.dataset]
        or folder.parent.name != manifest.source
        or folder.name != str(manifest.snapshotId)
    ):
        raise Error(400, "SOURCE_SCOPE_MISMATCH")
    if (
        manifest.sourceAsOf > now
        or manifest.publishedAt < manifest.sourceAsOf
        or manifest.publishedAt > now
        or manifest.expiresAt <= now
        or manifest.expiresAt > manifest.publishedAt + timedelta(days=1)
    ):
        raise Error(400, "INVALID_SOURCE_TIME")
    if manifest.dataset == "completed_baskets":
        window = manifest.window
        if (
            window is None
            or window.to > manifest.sourceAsOf
            or window.to - window.from_ != timedelta(days=90)
        ):
            raise Error(400, "INVALID_BASKET_WINDOW")
    elif manifest.window is not None:
        raise Error(400, "INVALID_SOURCE_WINDOW")
    records, previous, total = [], "", 0
    checksum = hashlib.sha256()
    cap = 100_000 if manifest.dataset == "completed_baskets" else 1000
    with data_file.open("rb") as handle:
        for line in iter(lambda: handle.readline(2 * 1024 * 1024 + 1), b""):
            if len(line) > 2 * 1024 * 1024:
                raise Error(400, "SOURCE_RECORD_TOO_LARGE")
            total += len(line)
            if total > FILE_LIMIT or len(records) >= cap:
                raise Error(400, "EXPORT_BUDGET_EXCEEDED")
            try:
                record = MODELS[manifest.dataset].model_validate_json(line)
                raw_record = json.loads(line)
            except (ValidationError, ValueError) as exc:
                raise Error(400, "INVALID_SOURCE_RECORD") from exc
            key = record.recordKey
            identity = str(record.orderId) if isinstance(record, Basket) else record.skuCode
            prefix = "order:" if isinstance(record, Basket) else "sku:"
            if key != prefix + identity or key <= previous or line != canonical(raw_record) + b"\n":
                raise Error(400, "INVALID_RECORD_ORDER_OR_ENCODING")
            if isinstance(record, Sellability) and (
                record.warehouseId != manifest.warehouseId
                or record.observedAt != manifest.sourceAsOf
                or record.eligibilityValidUntil > manifest.sourceAsOf + timedelta(seconds=120)
                or record.eligibilityValidUntil <= manifest.sourceAsOf
            ):
                raise Error(400, "INVALID_SELLABILITY")
            if isinstance(record, Basket):
                skus = [r.skuCode for r in record.items]
                if (
                    record.warehouseId != manifest.warehouseId
                    or not manifest.window.from_ <= record.completedAt < manifest.window.to
                    or skus != sorted(set(skus))
                    or bool(record.customerPrincipalId) != bool(record.identityMappingVersion)
                    or (
                        record.identityMappingVersion is not None
                        and record.identityMappingVersion != config.identity_mapping_version
                    )
                ):
                    raise Error(400, "INVALID_BASKET")
            previous = key
            checksum.update(line)
            records.append(record.model_dump(mode="json"))
    if checksum.hexdigest() != manifest.dataSha256 or len(records) != manifest.recordCount:
        raise Error(400, "EXPORT_CHECKSUM_MISMATCH")
    return manifest, records


def sync_once(store, now=None):
    now = now or datetime.now(UTC)
    results = []
    for owner in OWNERS.values():
        folder = store.config.export_root / owner
        if not folder.exists() or folder.is_symlink():
            continue
        for snapshot in sorted(folder.iterdir()):
            if snapshot.name.startswith(".") or not snapshot.is_dir():
                continue
            try:
                manifest, records = read_snapshot(snapshot, store.config, now)
                result = store.import_snapshot(manifest, records, now)
                IMPORTS.labels(owner, result).inc()
                results.append({"snapshotId": str(manifest.snapshotId), "status": result})
            except (Error, OSError) as exc:
                code = exc.code if isinstance(exc, Error) else "EXPORT_IO_UNAVAILABLE"
                log.warning("Recommendation intake source=%s code=%s", owner, code)
                IMPORTS.labels(owner, code).inc()
                results.append({"source": owner, "status": code})
    with store.connect() as conn:
        sources = store.sources(conn, baskets=False)
        for dataset, owner in OWNERS.items():
            row = sources.get(dataset)
            SOURCE_AS_OF.labels(owner, store.config.scope).set(
                row["source_as_of"].timestamp() if row else 0
            )
    return results
