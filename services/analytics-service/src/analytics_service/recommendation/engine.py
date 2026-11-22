import math
from collections import Counter, defaultdict
from datetime import datetime, timedelta
from fractions import Fraction
from itertools import combinations

from ..events import TZ

ALGORITHM_VERSION = "recommendation-v1"
MAX_PAIR_UPDATES = 5_000_000


class BudgetExceeded(RuntimeError):
    pass


def timestamp(value):
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def pairs(baskets, catalog):
    eligible = [b for b in baskets if b["eligible"]]
    if sum(len(b["items"]) * (len(b["items"]) - 1) // 2 for b in eligible) > MAX_PAIR_UPDATES:
        raise BudgetExceeded("Basket pair budget exceeded")
    counts, joint = Counter(), Counter()
    for basket in eligible:
        skus = sorted({item["skuCode"] for item in basket["items"]})
        counts.update(skus)
        joint.update(combinations(skus, 2))
    neighbors = defaultdict(list)
    total = len(eligible)
    for (a, b), common in joint.items():
        if (
            total < 20
            or common < 5
            or common * total <= counts[a] * counts[b]
            or a not in catalog
            or b not in catalog
            or catalog[a]["productId"] == catalog[b]["productId"]
        ):
            continue
        neighbors[a].append({"sku": b, "common": common, "anchorCount": counts[a]})
        neighbors[b].append({"sku": a, "common": common, "anchorCount": counts[b]})
    for anchor in neighbors:
        neighbors[anchor].sort(
            key=lambda r: (-Fraction(r["common"], r["anchorCount"]), -r["common"], r["sku"])
        )
        neighbors[anchor] = neighbors[anchor][:100]
    return {"total": total, "neighbors": dict(neighbors)}


def history(events, baskets, consents, now):
    """Keep capped, purpose-approved signals with an effective-time boundary."""
    cutoff = now - timedelta(days=30)
    capped = {}
    for event in events:
        consent = consents.get(event["subject_key"])
        at = event["effective_at"]
        if (
            not consent
            or not consent["enabled"]
            or event["consent_version"] != consent["version"]
            or at < max(cutoff, consent["granted_at"])
            or at > now
        ):
            continue
        key = (
            event["subject_key"],
            event["sku_code"],
            event["event_type"],
            at.astimezone(TZ).date(),
        )
        capped[key] = max(capped.get(key, at), at)
    signals = []
    for (subject, sku, kind, _), at in capped.items():
        signals.append((subject, sku, at, 1 if kind == "VIEW_ITEM" else 3))
    for basket in baskets:
        subject = basket.get("subjectKey")
        consent = consents.get(subject)
        at = timestamp(basket["completedAt"])
        if (
            not basket["eligible"]
            or not consent
            or not consent["enabled"]
            or basket.get("consentVersion") != consent["version"]
            or at < max(cutoff, consent["granted_at"])
            or at > now
        ):
            continue
        signals.extend((subject, item["skuCode"], at, 5) for item in basket["items"])
    profiles = defaultdict(Counter)
    for subject, sku, at, weight in signals:
        age = (now - at).total_seconds() / 86400
        profiles[subject][sku] += weight * 2 ** (-age / 7)
    return {subject: dict(weights.most_common(100)) for subject, weights in profiles.items()}


def collaborative(profiles):
    if sum(len(p) * (len(p) - 1) // 2 for p in profiles.values()) > MAX_PAIR_UPDATES:
        raise BudgetExceeded("Collaborative pair budget exceeded")
    counts, common = Counter(), Counter()
    for weights in profiles.values():
        skus = sorted(weights)
        counts.update(skus)
        common.update(combinations(skus, 2))
    result = defaultdict(list)
    for (a, b), overlap in common.items():
        if overlap >= 3:
            score = overlap / math.sqrt(counts[a] * counts[b]) * overlap / (overlap + 10)
            result[a].append({"sku": b, "score": score})
            result[b].append({"sku": a, "score": score})
    return {
        sku: sorted(rows, key=lambda r: (-r["score"], r["sku"]))[:100]
        for sku, rows in result.items()
    }


def eligible_catalog(catalog, inventory, now):
    stock = {row["skuCode"]: row for row in inventory}
    result = {}
    for row in catalog:
        item = stock.get(row["skuCode"])
        if (
            row["publiclyVisible"]
            and row["skuActive"]
            and row["priceVnd"] is not None
            and row["slug"]
            and row["name"]
            and item
            and item["skuActive"]
            and item["sellableUnits"] > 0
            and timestamp(item["eligibilityValidUntil"]) > now
        ):
            result[row["skuCode"]] = row
    return result


def similarity(anchor, candidate):
    score = 2 * bool(
        anchor["packagingType"] and anchor["packagingType"] == candidate["packagingType"]
    )
    for field in ("priceVnd", "weightGrams"):
        a, b = anchor[field], candidate[field]
        if a and b and min(a, b) * 4 >= max(a, b) * 3:
            score += 1
    return score


def ranked_candidates(placement, catalog, model, context, weights, stock, sales):
    """Return ordered (SKU, algorithm, reason) groups before final eligibility filtering."""

    def fallback(sku):
        return (catalog[sku].get("fallbackRank") or 2**63, sku)

    backup = [
        (sku, "catalog_fallback_v1", "APPROVED_FALLBACK")
        for sku in sorted(catalog, key=fallback)
        if catalog[sku].get("fallbackRank") is not None
    ]
    if placement == "product_detail":
        product = context["productId"]
        seed = next((r for r in catalog.values() if r["productId"] == product), None)
        if seed is None:
            return [], "PRODUCT_NOT_FOUND"
        anchor = catalog.get(context.get("skuCode"))
        groups = defaultdict(list)
        for sku, item in catalog.items():
            if (
                seed["categoryId"]
                and item["categoryId"] == seed["categoryId"]
                and item["productId"] != product
                and sku in stock
            ):
                groups[similarity(anchor, item) if anchor else 0].append(sku)
        related = []
        for score in sorted(groups, reverse=True):
            skus = groups[score]
            order = fallback
            if all(sku in sales for sku in skus):

                def order(sku):
                    return (-sales[sku], *fallback(sku))

            related.extend(
                (sku, "related_content_v1", "SIMILAR_ATTRIBUTES") for sku in sorted(skus, key=order)
            )
        return related + backup, None if anchor else "NO_SELECTED_SKU"
    if placement == "cart":
        scores = {}
        for anchor in context.get("skuCodes", []):
            for row in model.get("basket", {}).get("neighbors", {}).get(anchor, []):
                score = (Fraction(row["common"], row["anchorCount"]), row["common"])
                if score > scores.get(row["sku"], (Fraction(0), 0)):
                    scores[row["sku"]] = score
        ordered = sorted(scores, key=lambda s: (-scores[s][0], -scores[s][1], s))
        reason = None if ordered else "NO_QUALIFIED_PAIR"
        if model.get("basket", {}).get("total", 0) < 20:
            reason = "INSUFFICIENT_BASKETS"
        return [(s, "basket_pairs_v1", "BOUGHT_TOGETHER") for s in ordered] + backup, reason
    scores, categories = Counter(), Counter()
    for sku, weight in weights.items():
        if sku in catalog and catalog[sku]["categoryId"]:
            categories[catalog[sku]["categoryId"]] += weight
        for row in model.get("collaborative", {}).get(sku, []):
            scores[row["sku"]] += weight * row["score"]
    collaborative_rows = [
        (sku, "item_cosine_v1", "SIMILAR_CUSTOMER_INTERACTIONS")
        for sku in sorted(scores, key=lambda s: (-scores[s], s))
    ]
    affinity = [s for s in catalog if categories[catalog[s]["categoryId"]] > 0]
    affinity.sort(key=lambda s: (-categories[catalog[s]["categoryId"]], *fallback(s)))
    return (
        collaborative_rows
        + [(s, "category_affinity_v1", "RECENT_CATEGORY_INTEREST") for s in affinity]
        + backup,
        None if weights else "NO_HISTORY",
    )


def select(rows, eligible, excluded_products, limit):
    result, products = [], set(excluded_products)
    for sku, algorithm, reason in rows:
        item = eligible.get(sku)
        if item is None or item["productId"] in products:
            continue
        products.add(item["productId"])
        result.append(
            {
                **{
                    key: item[key]
                    for key in ("productId", "skuCode", "slug", "name", "thumbnailUrl", "priceVnd")
                },
                "rank": len(result) + 1,
                "algorithmVersion": algorithm,
                "reasonCode": reason,
            }
        )
        if len(result) == limit:
            break
    return result
