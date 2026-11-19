from datetime import datetime, timedelta
from decimal import ROUND_HALF_UP, Decimal

from .config import COHORT
from .snapshots import ORDER_METRICS

DAY = timedelta(days=1)
DEFINITIONS = {
    "orders_paid": ("ORDER", ["order-service"], "Số đơn ghi nhận mốc PAID đầu tiên trong kỳ."),
    "orders_completed": ("ORDER", ["order-service"], "Số đơn ghi nhận mốc COMPLETED trong kỳ."),
    "active_customers": ("CUSTOMER", ["order-service", "profile-service"], None),
    "sellable_stock": ("ITEM", ["inventory-service"], None),
    "recognized_revenue": ("VND", ["finance-service"], None),
    "profit": ("VND", ["finance-service"], None),
}


def days(start, end):
    return [start + index * DAY for index in range((end - start).days)]


def observation(value, quality, reasons):
    return {
        "value": str(value) if value is not None else None,
        "quality": quality,
        "reasonCodes": reasons,
    }


def metadata(snapshot, start, end):
    return {
        "snapshotId": str(snapshot["generation_id"]),
        "snapshotPublishedAt": snapshot["published_at"],
        "timeZone": "Asia/Ho_Chi_Minh",
        "from": start,
        "to": end,
        "orderCohorts": [COHORT],
    }


def definition(metric, as_of):
    unit, sources, label = DEFINITIONS[metric]
    return {
        "metricId": metric,
        "unit": unit,
        "sources": sources,
        "definition": label,
        "definitionVersion": "1" if label else None,
        "sourceAsOf": as_of,
    }


def bucket(snapshot, metric, day, counts, now, freshness):
    coverage = snapshot["coverage"]
    approved = coverage["days"].get(day.isoformat())
    if approved is None:
        reason = "SOURCE_INVALID" if coverage["sourceInvalid"] else "COVERAGE_GAP"
        return observation(None, "UNAVAILABLE", [reason]), None
    late = now - datetime.fromisoformat(approved["approvedAt"]) > freshness
    return (
        observation(
            counts.get((day, ORDER_METRICS[metric]), 0),
            "STALE" if late else "FRESH",
            ["SOURCE_LATE"] if late else [],
        ),
        datetime.fromisoformat(approved["sourceAsOf"]),
    )


def aggregate(snapshot, metric, start, end, counts, now, freshness):
    if metric not in ORDER_METRICS:
        reason = (
            "METRIC_APPROVAL_PENDING"
            if metric in ("recognized_revenue", "profit")
            else "SOURCE_CONTRACT_PENDING"
        )
        return {
            **observation(None, "UNAVAILABLE", [reason]),
            "coveredPeriods": [],
            "sourceAsOf": None,
        }
    periods = []
    reasons = set()
    total = 0
    sources = []
    for day in days(start, end):
        result, as_of = bucket(snapshot, metric, day, counts, now, freshness)
        reasons.update(result["reasonCodes"])
        if as_of is None:
            continue
        total += int(result["value"])
        sources.append(as_of)
        if periods and periods[-1]["to"] == day:
            periods[-1]["to"] += DAY
        else:
            periods.append({"from": day, "to": day + DAY})
    if not sources:
        quality = "UNAVAILABLE"
    elif len(sources) < (end - start).days:
        quality = "PARTIAL"
    else:
        quality = "STALE" if reasons else "FRESH"
    return {
        **observation(total if sources else None, quality, sorted(reasons)),
        "coveredPeriods": periods,
        "sourceAsOf": max(sources, default=None),
    }


def comparison(current, previous, start, end):
    delta = percent = None
    if current["quality"] == previous["quality"] == "FRESH":
        baseline = int(previous["value"])
        delta = int(current["value"]) - baseline
        if baseline:
            percent = str(
                (Decimal(delta) * 100 / abs(baseline)).quantize(
                    Decimal(".01"), rounding=ROUND_HALF_UP
                )
            )
    return {
        **previous,
        "from": start,
        "to": end,
        "absoluteChange": str(delta) if delta is not None else None,
        "percentChange": percent,
    }


def dashboard(snapshot, start, end, compare, counts, now, freshness):
    cards = []
    for metric in DEFINITIONS:
        current = aggregate(snapshot, metric, start, end, counts, now, freshness)
        card = {**definition(metric, current["sourceAsOf"]), **current}
        if compare:
            prior_start = start - (end - start)
            previous = aggregate(snapshot, metric, prior_start, start, counts, now, freshness)
            card["comparison"] = comparison(current, previous, prior_start, start)
        cards.append(card)
    return {**metadata(snapshot, start, end), "metrics": cards}


def series(snapshot, metric, start, end, counts, now, freshness):
    buckets = []
    latest = None
    for day in days(start, end):
        result, as_of = bucket(snapshot, metric, day, counts, now, freshness)
        buckets.append({"date": day, **result})
        if as_of is not None:
            latest = max(latest, as_of) if latest else as_of
    return {**metadata(snapshot, start, end), **definition(metric, latest), "buckets": buckets}
