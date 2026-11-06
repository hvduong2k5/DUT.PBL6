from datetime import UTC, date, datetime, time, timedelta
from decimal import ROUND_HALF_UP, Decimal
from typing import Annotated, Literal

from psycopg.types.json import Jsonb
from pydantic import (
    AwareDatetime,
    BaseModel,
    ConfigDict,
    Field,
    StringConstraints,
    TypeAdapter,
    ValidationError,
)

from . import db
from .events import TZ, canonical, digest
from .report import DAY, days

VERSION = "1"
BASELINE_DAYS = 14
MIN_IN_STOCK_DAYS = 7
MIN_BUSINESS_DATE = date(2000, 1, 1)
MAX_BUSINESS_DATE = date(2100, 12, 31)
WORKBENCH_SKUS = 100
WORKBENCH_BATCHES = 200
CODE_PATTERN = r"^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$"
Code = Annotated[str, StringConstraints(pattern=CODE_PATTERN)]
Quantity = Annotated[int, Field(strict=True, ge=0, le=2_147_483_647)]
BusinessDate = Annotated[date, Field(strict=True, ge=MIN_BUSINESS_DATE, le=MAX_BUSINESS_DATE)]


class Export(BaseModel):
    model_config = ConfigDict(extra="forbid")

    schemaVersion: Literal["1"]
    skuCode: Code
    warehouseId: Code
    businessDate: BusinessDate
    revision: Annotated[int, Field(strict=True, ge=1, le=2_147_483_647)]
    sourceAsOf: Annotated[AwareDatetime, Field(strict=True)]
    approvedBy: Annotated[
        str, StringConstraints(strip_whitespace=True, min_length=1, max_length=128)
    ]
    evidenceRef: Annotated[
        str, StringConstraints(strip_whitespace=True, min_length=1, max_length=256)
    ]


class OrderDay(Export):
    kind: Literal["order_day"]
    source: Literal["order-service"]
    soldUnits: Quantity
    returnedUnits: Quantity


class InventoryDay(Export):
    kind: Literal["inventory_day"]
    source: Literal["inventory-service"]
    stockoutMinutes: Annotated[int, Field(strict=True, ge=0, le=1440)]


class Batch(BaseModel):
    model_config = ConfigDict(extra="forbid")

    batchCode: Code
    expiresOn: BusinessDate
    sellableUnits: Annotated[int, Field(strict=True, ge=1, le=2_147_483_647)]


class InventoryBatches(Export):
    kind: Literal["inventory_batches"]
    source: Literal["inventory-service"]
    batches: Annotated[list[Batch], Field(max_length=2000)]


EXPORT = TypeAdapter(
    Annotated[OrderDay | InventoryDay | InventoryBatches, Field(discriminator="kind")]
)


class SourceNotYetCurrent(db.Conflict):
    pass


def parse_export(body, now):
    try:
        export = EXPORT.validate_json(body)
    except ValidationError as exc:
        reasons = ", ".join(sorted({item["type"] for item in exc.errors(include_input=False)}))
        raise db.Conflict(f"Invalid supply export ({reasons})") from None
    if export.sourceAsOf > now:
        if export.sourceAsOf - now <= timedelta(minutes=5):
            raise SourceNotYetCurrent("Supply export source clock is ahead")
        raise db.Conflict("Supply export sourceAsOf is in the future")
    try:
        local_as_of = export.sourceAsOf.astimezone(TZ)
    except OverflowError:
        raise db.Conflict("Supply export sourceAsOf is outside the supported date range") from None
    if isinstance(export, InventoryBatches):
        if local_as_of.date() != export.businessDate:
            raise db.Conflict("Batch snapshot businessDate must match sourceAsOf in Viet Nam")
        if len({batch.batchCode for batch in export.batches}) != len(export.batches):
            raise db.Conflict("Duplicate batch code")
        if any(batch.expiresOn <= export.businessDate for batch in export.batches):
            raise db.Conflict("A sellable batch must expire after the snapshot date")
        export.batches.sort(key=lambda batch: batch.batchCode)
    elif local_as_of < datetime.combine(export.businessDate + DAY, time.min, TZ):
        raise db.Conflict("Daily exports require a closed business day")
    return export


def import_export(conn, settings, export):
    identity = (
        settings.scope,
        settings.pipeline,
        export.skuCode,
        export.warehouseId,
        export.kind,
        export.businessDate,
    )
    source_as_of = export.sourceAsOf.astimezone(UTC)
    content = export.model_dump(mode="json")
    content["sourceAsOf"] = source_as_of.isoformat()
    fingerprint = digest(canonical(content))
    # Serialize revisions of a single source/day so retries and concurrent imports agree.
    lock_key = [str(item) for item in identity]
    conn.execute(
        "SELECT pg_advisory_xact_lock(hashtextextended(%s, 13))", (canonical(lock_key).decode(),)
    )
    existing = conn.execute(
        """SELECT content_digest FROM supply_export
           WHERE (scope_key,pipeline,sku_code,warehouse_id,kind,business_date,revision)
                 =(%s,%s,%s,%s,%s,%s,%s)""",
        (*identity, export.revision),
    ).fetchone()
    if existing:
        if existing["content_digest"] == fingerprint:
            return "duplicate"
        raise db.Conflict("Supply export content changed at the same revision")
    previous = conn.execute(
        """SELECT revision, source_as_of FROM supply_export
           WHERE (scope_key,pipeline,sku_code,warehouse_id,kind,business_date)=(%s,%s,%s,%s,%s,%s)
           ORDER BY revision DESC LIMIT 1""",
        identity,
    ).fetchone()
    if previous and (
        export.revision <= previous["revision"] or source_as_of < previous["source_as_of"]
    ):
        raise db.Conflict("Supply export revision precedes the current source revision")
    payload = export.model_dump(
        mode="json", exclude=Export.model_fields.keys() | {"source", "kind"}
    )
    conn.execute(
        """INSERT INTO supply_export
           (scope_key,pipeline,sku_code,warehouse_id,kind,business_date,revision,
            source_as_of,approved_by,evidence_ref,payload,content_digest)
           VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)""",
        (
            *identity,
            export.revision,
            source_as_of,
            export.approvedBy,
            export.evidenceRef,
            Jsonb(payload),
            fingerprint,
        ),
    )
    return "imported"


def read_days(conn, settings, sku, warehouse, start, end):
    rows = conn.execute(
        """SELECT DISTINCT ON (kind,business_date) kind,business_date,payload,source_as_of,revision
           FROM supply_export WHERE scope_key=%s AND pipeline=%s AND sku_code=%s AND warehouse_id=%s
             AND kind IN ('order_day','inventory_day') AND business_date >= %s AND business_date < %s
           ORDER BY kind,business_date,revision DESC""",
        (settings.scope, settings.pipeline, sku, warehouse, start, end),
    )
    return {(row["kind"], row["business_date"]): row for row in rows}


def read_batches(conn, settings, sku, warehouse):
    return conn.execute(
        """SELECT business_date,payload,source_as_of,revision FROM supply_export
           WHERE scope_key=%s AND pipeline=%s AND sku_code=%s AND warehouse_id=%s
             AND kind='inventory_batches'
           ORDER BY business_date DESC,revision DESC LIMIT 1""",
        (settings.scope, settings.pipeline, sku, warehouse),
    ).fetchone()


def rounded(numerator, denominator):
    return str((Decimal(numerator) / denominator).quantize(Decimal(".01"), rounding=ROUND_HALF_UP))


def velocity(rows, start, end):
    sales = []
    stock = []
    series = []
    for day in days(start, end):
        order = rows.get(("order_day", day))
        inventory = rows.get(("inventory_day", day))
        series.append(
            {
                "date": day,
                "soldUnits": order["payload"]["soldUnits"] if order else None,
                "returnedUnits": order["payload"]["returnedUnits"] if order else None,
                "stockoutMinutes": inventory["payload"]["stockoutMinutes"] if inventory else None,
                "orderRevision": order["revision"] if order else None,
                "inventoryRevision": inventory["revision"] if inventory else None,
            }
        )
        if order:
            sales.append(order)
        if inventory:
            stock.append(inventory)
    sold = sum(row["payload"]["soldUnits"] for row in sales)
    returned = sum(row["payload"]["returnedUnits"] for row in sales)
    expected = (end - start).days
    status = "UNAVAILABLE"
    if sales:
        status = "READY" if len(sales) == len(stock) == expected else "PARTIAL"
    return {
        "from": start,
        "to": end,
        "status": status,
        "soldUnits": sold if sales else None,
        "returnedUnits": returned if sales else None,
        "netUnits": sold - returned if sales else None,
        "unitsPerDay": rounded(sold - returned, len(sales)) if sales else None,
        "salesCoveredDays": len(sales),
        "inventoryCoveredDays": len(stock),
        "salesAsOf": max((row["source_as_of"] for row in sales), default=None),
        "inventoryAsOf": max((row["source_as_of"] for row in stock), default=None),
        "stockoutDays": sum(row["payload"]["stockoutMinutes"] > 0 for row in stock)
        if stock
        else None,
        "series": series,
    }


def allocate_fefo(batches, start, end, demand, observed_days):
    ordered = sorted(batches, key=lambda batch: (batch["expiresOn"], batch["batchCode"]))
    remaining = {batch["batchCode"]: batch["sellableUnits"] for batch in ordered}
    expiry = {batch["batchCode"]: date.fromisoformat(batch["expiresOn"]) for batch in ordered}
    index = 0
    # Demand accrues across whole days; integer allocation preserves units across batches.
    for elapsed, day in enumerate(days(start + DAY, end), start=1):
        needed = demand * elapsed // observed_days - demand * (elapsed - 1) // observed_days
        while needed > 0 and index < len(ordered):
            batch_id = ordered[index]["batchCode"]
            if expiry[batch_id] <= day or remaining[batch_id] == 0:
                index += 1
                continue
            taken = min(needed, remaining[batch_id])
            remaining[batch_id] -= taken
            needed -= taken
    return [
        {**batch, "unitsAtRisk": remaining[batch["batchCode"]]}
        for batch in ordered
        if expiry[batch["batchCode"]] <= end
    ]


def expiry_risk(snapshot, rows, horizon, now, freshness):
    result = {
        "status": "UNAVAILABLE",
        "reasonCode": "BATCH_SNAPSHOT_MISSING",
        "snapshotDate": None,
        "sourceAsOf": None,
        "snapshotRevision": None,
        "baselineDays": BASELINE_DAYS,
        "inStockDays": 0,
        "demandUnitsPerDay": None,
        "batches": [],
    }
    if snapshot is None:
        return result
    start = snapshot["business_date"]
    result.update(
        snapshotDate=start,
        sourceAsOf=snapshot["source_as_of"],
        snapshotRevision=snapshot["revision"],
    )
    if now - snapshot["source_as_of"] > freshness:
        return {**result, "reasonCode": "BATCH_SNAPSHOT_STALE"}
    candidates = snapshot["payload"]["batches"]
    if not any(
        date.fromisoformat(batch["expiresOn"]) <= start + horizon * DAY for batch in candidates
    ):
        return {**result, "status": "READY", "reasonCode": None}
    baseline = days(start - BASELINE_DAYS * DAY, start)
    if any((kind, day) not in rows for day in baseline for kind in ("order_day", "inventory_day")):
        return {**result, "reasonCode": "BASELINE_COVERAGE_GAP"}
    in_stock = [
        day for day in baseline if rows["inventory_day", day]["payload"]["stockoutMinutes"] == 0
    ]
    result["inStockDays"] = len(in_stock)
    if len(in_stock) < MIN_IN_STOCK_DAYS:
        return {**result, "reasonCode": "INSUFFICIENT_IN_STOCK_DAYS"}
    gross = sum(rows["order_day", day]["payload"]["soldUnits"] for day in in_stock)
    return {
        **result,
        "status": "READY",
        "reasonCode": None,
        "demandUnitsPerDay": rounded(gross, len(in_stock)),
        "batches": allocate_fefo(candidates, start, start + horizon * DAY, gross, len(in_stock)),
    }


def workbench(conn, settings, warehouse, start, end, horizon, now, freshness):
    scope = (settings.scope, settings.pipeline, warehouse)
    skus = [
        row["sku_code"]
        for row in conn.execute(
            """SELECT DISTINCT sku_code FROM supply_export
               WHERE scope_key=%s AND pipeline=%s AND warehouse_id=%s
               ORDER BY sku_code LIMIT %s""",
            (*scope, WORKBENCH_SKUS + 1),
        )
    ]
    truncated = len(skus) > WORKBENCH_SKUS
    skus = skus[:WORKBENCH_SKUS]
    snapshots = {
        row["sku_code"]: row
        for row in conn.execute(
            """SELECT DISTINCT ON (sku_code) sku_code,business_date,payload,source_as_of,revision
               FROM supply_export WHERE scope_key=%s AND pipeline=%s AND warehouse_id=%s
                 AND sku_code=ANY(%s) AND kind='inventory_batches'
               ORDER BY sku_code,business_date DESC,revision DESC""",
            (*scope, skus),
        )
    }
    fresh_days = [
        row["business_date"] for row in snapshots.values() if now - row["source_as_of"] <= freshness
    ]
    baseline_first = min((day - BASELINE_DAYS * DAY for day in fresh_days), default=start)
    baseline_end = max(fresh_days, default=start)
    days_by_sku = {sku: {} for sku in skus}
    for row in conn.execute(
        """SELECT DISTINCT ON (sku_code,kind,business_date)
                  sku_code,kind,business_date,payload,source_as_of,revision
           FROM supply_export WHERE scope_key=%s AND pipeline=%s AND warehouse_id=%s
             AND sku_code=ANY(%s) AND kind IN ('order_day','inventory_day')
             AND ((business_date >= %s AND business_date < %s)
               OR (business_date >= %s AND business_date < %s))
           ORDER BY sku_code,kind,business_date,revision DESC""",
        (*scope, skus, start, end, baseline_first, baseline_end),
    ):
        days_by_sku[row["sku_code"]][row["kind"], row["business_date"]] = row
    ranking = []
    batches = []
    for sku in skus:
        rows = days_by_sku[sku]
        period = velocity(rows, start, end)
        risk = expiry_risk(snapshots.get(sku), rows, horizon, now, freshness)
        ranking.append(
            {
                "skuCode": sku,
                "status": period["status"],
                "netUnits": period["netUnits"],
                "unitsPerDay": period["unitsPerDay"],
                "salesCoveredDays": period["salesCoveredDays"],
                "inventoryCoveredDays": period["inventoryCoveredDays"],
                "stockoutDays": period["stockoutDays"],
                "riskReason": risk["reasonCode"],
                "snapshotAsOf": risk["sourceAsOf"],
            }
        )
        batches.extend(
            {"skuCode": sku, **batch} for batch in risk["batches"] if batch["unitsAtRisk"] > 0
        )
    ranking.sort(
        key=lambda row: (
            row["status"] != "READY",
            row["unitsPerDay"] is None,
            -Decimal(row["unitsPerDay"] or "0"),
            row["skuCode"],
        )
    )
    batches.sort(
        key=lambda row: (row["expiresOn"], -row["unitsAtRisk"], row["skuCode"], row["batchCode"])
    )
    return {
        "warehouseId": warehouse,
        "from": start,
        "to": end,
        "horizonDays": horizon,
        "ranking": ranking,
        "skuLimitReached": truncated,
        "batches": batches[:WORKBENCH_BATCHES],
        "batchCount": len(batches),
    }
