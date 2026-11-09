import hashlib
import hmac
from contextlib import contextmanager
from datetime import UTC, timedelta
from uuid import uuid4

import psycopg
from prometheus_client import Histogram
from psycopg.rows import dict_row
from psycopg.types.json import Jsonb

from ..events import canonical
from . import engine

BUILD_SECONDS = Histogram("analytics_recommendation_build_seconds", "Generation build time")


class Error(Exception):
    def __init__(self, status, code):
        self.status = status
        self.code = code
        super().__init__(code)


class Store:
    def __init__(self, config):
        self.config = config

    @contextmanager
    def connect(self):
        with psycopg.connect(
            self.config.dsn,
            row_factory=dict_row,
            connect_timeout=2,
            options="-c search_path=analytics_recommendation,public -c statement_timeout=5000",
        ) as conn:
            yield conn

    def subject(self, principal):
        hashed = hmac.new(
            self.config.subject_key, str(principal).encode(), hashlib.sha256
        ).hexdigest()
        return f"{self.config.key_version}:{hashed}"

    def lock_privacy(self, conn):
        return conn.execute("SELECT epoch FROM privacy_state WHERE id=1 FOR UPDATE").fetchone()[
            "epoch"
        ]

    def consent(self, conn, subject, lock=False):
        suffix = " FOR UPDATE" if lock else ""
        return conn.execute(
            "SELECT * FROM consent WHERE subject_key=%s" + suffix, (subject,)
        ).fetchone()

    @staticmethod
    def consent_response(row):
        return {
            "enabled": row["enabled"] if row else False,
            "consentVersion": row["version"] if row else 0,
            "policyVersion": "personalization-v1",
            "grantedAt": row["granted_at"].isoformat() if row and row["granted_at"] else None,
            "revokedAt": row["revoked_at"].isoformat() if row and row["revoked_at"] else None,
        }

    def get_consent(self, subject):
        with self.connect() as conn:
            return self.consent_response(self.consent(conn, subject))

    def set_consent(self, subject, body, expected, request_key, now):
        fingerprint = hashlib.sha256(canonical(body)).hexdigest()
        with self.connect() as conn:
            self.lock_privacy(conn)
            conn.execute(
                "INSERT INTO consent(subject_key) VALUES (%s) ON CONFLICT DO NOTHING", (subject,)
            )
            current = self.consent(conn, subject, lock=True)
            previous = conn.execute(
                "SELECT digest FROM consent_request WHERE subject_key=%s AND request_key=%s",
                (subject, request_key),
            ).fetchone()
            if previous:
                if previous["digest"] != fingerprint:
                    raise Error(409, "IDEMPOTENCY_CONFLICT")
                return self.consent_response(current)
            if current["version"] != expected:
                raise Error(412, "CONSENT_VERSION_CONFLICT")
            if current["enabled"] != body["enabled"]:
                conn.execute(
                    """UPDATE consent SET enabled=%s,version=version+1,
                       granted_at=CASE WHEN %s THEN %s ELSE granted_at END,
                       revoked_at=CASE WHEN %s THEN NULL ELSE %s END
                       WHERE subject_key=%s""",
                    (body["enabled"], body["enabled"], now, body["enabled"], now, subject),
                )
                if not body["enabled"]:
                    conn.execute("UPDATE privacy_state SET epoch=epoch+1 WHERE id=1")
                    conn.execute(
                        """INSERT INTO erasure VALUES(%s,%s,%s) ON CONFLICT(subject_key)
                           DO UPDATE SET requested_at=EXCLUDED.requested_at,
                           through_version=EXCLUDED.through_version""",
                        (subject, now, current["version"] + 1),
                    )
            conn.execute(
                "INSERT INTO consent_request VALUES(%s,%s,%s,%s)",
                (subject, request_key, fingerprint, now),
            )
            return self.consent_response(self.consent(conn, subject))

    def accept_event(self, subject, event, now):
        payload = event.model_dump(mode="json")
        payload["occurredAt"] = event.occurredAt.astimezone(UTC).isoformat()
        fingerprint = hashlib.sha256(canonical(payload)).hexdigest()
        with self.connect() as conn:
            current = self.consent(conn, subject, lock=True)
            if not current or not current["enabled"] or current["version"] != event.consentVersion:
                raise Error(409, "CONSENT_VERSION_CONFLICT")
            if event.occurredAt < max(
                now - timedelta(hours=24), current["granted_at"]
            ) or event.occurredAt > now + timedelta(minutes=5):
                raise Error(400, "INVALID_EVENT_TIME")
            previous = conn.execute(
                "SELECT digest FROM behavior_event WHERE subject_key=%s AND event_id=%s",
                (subject, event.eventId),
            ).fetchone()
            if previous:
                if previous["digest"] != fingerprint:
                    raise Error(409, "EVENT_ID_CONFLICT")
                return "DUPLICATE"
            snapshots = self.sources(conn, baskets=False)
            if not any(
                r["skuCode"] == event.skuCode
                for r in snapshots.get("catalog", {}).get("records", [])
            ):
                raise Error(400, "UNKNOWN_SKU")
            conn.execute(
                """INSERT INTO behavior_event VALUES
                   (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)""",
                (
                    subject,
                    event.eventId,
                    self.config.scope,
                    current["version"],
                    fingerprint,
                    event.eventType,
                    event.skuCode,
                    event.occurredAt,
                    min(now, event.occurredAt),
                    now,
                    event.recommendationId,
                ),
            )
        return "ACCEPTED"

    def import_snapshot(self, manifest, records, now):
        with self.connect() as conn:
            conn.execute(
                "SELECT pg_advisory_xact_lock(hashtextextended(%s,41))",
                (self.config.scope + manifest.dataset,),
            )
            self.lock_privacy(conn)
            existing = conn.execute(
                """SELECT snapshot_id,digest,manifest FROM source_snapshot WHERE snapshot_id=%s
                   OR (scope_id=%s AND dataset=%s AND source_revision=%s)""",
                (manifest.snapshotId, self.config.scope, manifest.dataset, manifest.sourceRevision),
            ).fetchall()
            if existing:
                if len(existing) != 1 or existing[0] != {
                    "snapshot_id": manifest.snapshotId,
                    "digest": manifest.dataSha256,
                    "manifest": manifest.model_dump(mode="json", by_alias=True),
                }:
                    raise Error(409, "SNAPSHOT_CONFLICT")
                return "DUPLICATE"
            current = conn.execute(
                """SELECT s.* FROM source_pointer p JOIN source_snapshot s USING(snapshot_id)
                   WHERE p.scope_id=%s AND p.dataset=%s""",
                (self.config.scope, manifest.dataset),
            ).fetchone()
            if current and (
                current["source_revision"] >= manifest.sourceRevision
                or current["source_as_of"] > manifest.sourceAsOf
            ):
                raise Error(409, "STALE_SOURCE_REVISION")
            if manifest.dataset == "completed_baskets":
                approved = {
                    row["subject_key"]: row
                    for row in conn.execute("SELECT * FROM consent WHERE enabled").fetchall()
                }
                for row in records:
                    principal = row.pop("customerPrincipalId")
                    subject = self.subject(principal) if principal else None
                    consent = approved.get(subject)
                    if consent and engine.timestamp(row["completedAt"]) >= consent["granted_at"]:
                        row["subjectKey"] = subject
                        row["consentVersion"] = consent["version"]
            conn.execute(
                """INSERT INTO source_snapshot
                   (snapshot_id,scope_id,dataset,source_revision,source_as_of,manifest,records,digest)
                   VALUES (%s,%s,%s,%s,%s,%s,%s,%s)""",
                (
                    manifest.snapshotId,
                    self.config.scope,
                    manifest.dataset,
                    manifest.sourceRevision,
                    manifest.sourceAsOf,
                    Jsonb(manifest.model_dump(mode="json", by_alias=True)),
                    Jsonb(records),
                    manifest.dataSha256,
                ),
            )
            conn.execute(
                """INSERT INTO source_pointer VALUES(%s,%s,%s)
                   ON CONFLICT(scope_id,dataset) DO UPDATE SET snapshot_id=EXCLUDED.snapshot_id""",
                (self.config.scope, manifest.dataset, manifest.snapshotId),
            )
        return "IMPORTED"

    def sources(self, conn, baskets=True):
        return {
            row["dataset"]: row
            for row in conn.execute(
                """SELECT s.snapshot_id,s.scope_id,s.dataset,s.source_revision,s.source_as_of,
                          s.manifest,s.digest,s.received_at,
                          CASE WHEN s.dataset='completed_baskets' AND NOT %s
                          THEN '[]'::jsonb ELSE s.records END AS records
                   FROM source_pointer p JOIN source_snapshot s USING(snapshot_id)
                   WHERE p.scope_id=%s""",
                (baskets, self.config.scope),
            ).fetchall()
        }

    @BUILD_SECONDS.time()
    def build(self, now):
        with self.connect() as conn:
            conn.execute("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY")
            epoch = conn.execute("SELECT epoch FROM privacy_state WHERE id=1").fetchone()["epoch"]
            sources = self.sources(conn)
            self.check_sources(sources, now)
            vector = {name: str(row["snapshot_id"]) for name, row in sources.items()}
            consents = {
                r["subject_key"]: r for r in conn.execute("SELECT * FROM consent").fetchall()
            }
            events = conn.execute(
                """SELECT * FROM behavior_event WHERE scope_id=%s AND effective_at >= %s
                   ORDER BY received_at,event_id LIMIT 1000001""",
                (self.config.scope, now - timedelta(days=30)),
            ).fetchall()
            if len(events) > 1_000_000:
                raise Error(503, "MODEL_BUDGET_EXCEEDED")
            fingerprint = hashlib.sha256(
                canonical(
                    {
                        "version": engine.ALGORITHM_VERSION,
                        "sources": vector,
                        "epoch": epoch,
                        "mapping": self.config.mapping_version,
                        "cutoff": int(now.timestamp()) // 300,
                        "events": [(str(e["event_id"]), e["digest"]) for e in events],
                        "fallback": self.config.fallback_ranks,
                    }
                )
            ).hexdigest()
            current = conn.execute(
                """SELECT g.* FROM serving_pointer p JOIN generation g USING(generation_id)
                   WHERE p.scope_id=%s""",
                (self.config.scope,),
            ).fetchone()
            if current and current["input_digest"] == fingerprint:
                return str(current["generation_id"])
            expected = current["generation_id"] if current else None
        catalog = {r["skuCode"]: r for r in sources["catalog"]["records"]}
        baskets = sources.get("completed_baskets", {}).get("records", [])
        profiles = engine.history(events, baskets, consents, now)
        try:
            payload = {
                "basket": engine.pairs(baskets, catalog),
                "collaborative": engine.collaborative(profiles),
            }
        except engine.BudgetExceeded as exc:
            raise Error(503, "MODEL_BUDGET_EXCEEDED") from exc
        with self.connect() as conn:
            conn.execute(
                "SELECT pg_advisory_xact_lock(hashtextextended(%s,42))", (self.config.scope,)
            )
            if self.lock_privacy(conn) != epoch:
                raise Error(409, "BUILD_INPUT_CHANGED")
            pointers = conn.execute(
                "SELECT dataset,snapshot_id FROM source_pointer WHERE scope_id=%s",
                (self.config.scope,),
            ).fetchall()
            if {r["dataset"]: str(r["snapshot_id"]) for r in pointers} != vector:
                raise Error(409, "BUILD_INPUT_CHANGED")
            actual = conn.execute(
                """SELECT g.* FROM serving_pointer p JOIN generation g USING(generation_id)
                                  WHERE p.scope_id=%s""",
                (self.config.scope,),
            ).fetchone()
            if actual and actual["input_digest"] == fingerprint:
                return str(actual["generation_id"])
            if (actual["generation_id"] if actual else None) != expected:
                raise Error(409, "BUILD_INPUT_CHANGED")
            generation = uuid4()
            conn.execute(
                "INSERT INTO generation VALUES(%s,%s,%s,%s,%s,%s,%s)",
                (
                    generation,
                    self.config.scope,
                    fingerprint,
                    epoch,
                    now,
                    Jsonb(vector),
                    Jsonb(payload),
                ),
            )
            conn.execute(
                """INSERT INTO serving_pointer VALUES(%s,%s) ON CONFLICT(scope_id)
                   DO UPDATE SET generation_id=EXCLUDED.generation_id""",
                (self.config.scope, generation),
            )
            return str(generation)

    def check_sources(self, sources, now):
        for name, max_age in (
            ("catalog", self.config.catalog_age),
            ("sellability", self.config.inventory_age),
        ):
            row = sources.get(name)
            if (
                not row
                or (now - row["source_as_of"]).total_seconds() > max_age
                or engine.timestamp(row["manifest"]["expiresAt"]) <= now
            ):
                raise Error(503, "RECOMMENDATION_SOURCE_UNAVAILABLE")

    def recommend(self, placement, context, limit, subject, now):
        with self.connect() as conn:
            # One coherent view covers consent, source pointers and generation selection.
            conn.execute("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
            sources = self.sources(conn, baskets=False)
            self.check_sources(sources, now)
            catalog = {r["skuCode"]: dict(r) for r in sources["catalog"]["records"]}
            for sku, rank in self.config.fallback_ranks.items():
                if sku in catalog:
                    catalog[sku]["fallbackRank"] = rank
            if placement == "product_detail":
                product = [
                    r
                    for r in catalog.values()
                    if r["productId"] == context["productId"] and r["publiclyVisible"]
                ]
                if not product:
                    raise Error(404, "PRODUCT_NOT_FOUND")
                selected = context.get("skuCode")
                if selected and (
                    selected not in catalog
                    or catalog[selected]["productId"] != context["productId"]
                    or not catalog[selected]["skuActive"]
                    or not catalog[selected]["publiclyVisible"]
                ):
                    raise Error(400, "INVALID_RECOMMENDATION_QUERY")
            eligible = engine.eligible_catalog(
                list(catalog.values()), sources["sellability"]["records"], now
            )
            generation = conn.execute(
                """SELECT g.* FROM serving_pointer p JOIN generation g USING(generation_id)
                   WHERE p.scope_id=%s""",
                (self.config.scope,),
            ).fetchone()
            model, reason = {}, None
            if generation and now - generation["built_at"] <= timedelta(
                seconds=self.config.model_age
            ):
                model = dict(generation["payload"])
            else:
                reason = "MODEL_NOT_READY" if generation is None else "STALE_MODEL"
            if model and any(
                generation["sources"].get(name) != str(sources[name]["snapshot_id"])
                for name in ("catalog", "completed_baskets")
                if name in sources
            ):
                model = {}
                reason = "SOURCE_CHANGED_REBUILD"
            order = sources.get("completed_baskets")
            if not order or now - order["source_as_of"] > timedelta(seconds=self.config.order_age):
                model.pop("basket", None)
                model.pop("collaborative", None)
                if placement == "cart":
                    reason = "STALE_ORDER_SOURCE"
            epoch = conn.execute("SELECT epoch FROM privacy_state WHERE id=1").fetchone()["epoch"]
            weights = {}
            if placement == "home" and subject and self.config.personalized:
                consent = self.consent(conn, subject)
                if consent and consent["enabled"]:
                    events = conn.execute(
                        """SELECT * FROM behavior_event WHERE scope_id=%s AND subject_key=%s
                           AND effective_at >= %s ORDER BY received_at LIMIT 100001""",
                        (self.config.scope, subject, now - timedelta(days=30)),
                    ).fetchall()
                    baskets = []
                    if order and now - order["source_as_of"] <= timedelta(
                        seconds=self.config.order_age
                    ):
                        baskets = conn.execute(
                            """SELECT jsonb_path_query_array(records,
                               '$[*] ? (@.subjectKey == $subject)', %s) AS records
                               FROM source_snapshot WHERE snapshot_id=%s""",
                            (Jsonb({"subject": subject}), order["snapshot_id"]),
                        ).fetchone()["records"]
                    weights = engine.history(events, baskets, {subject: consent}, now).get(
                        subject, {}
                    )
                    if generation is None or generation["privacy_epoch"] != epoch:
                        model.pop("collaborative", None)
                        reason = "PRIVACY_REBUILD"
                else:
                    reason = "PERSONALIZATION_NOT_ENABLED"
            elif placement == "home":
                reason = "PERSONALIZATION_NOT_ENABLED"
            if placement == "cart" and not context.get("skuCodes"):
                rows, ranking_reason = [], None
            else:
                rows, ranking_reason = engine.ranked_candidates(
                    placement, catalog, model, context, weights, eligible, self.sales(conn, now)
                )
            excluded = set()
            if placement == "product_detail":
                excluded.add(context["productId"])
            if placement == "cart":
                excluded.update(
                    catalog[s]["productId"] for s in context["skuCodes"] if s in catalog
                )
            items = engine.select(rows, eligible, excluded, limit)
            algorithms = {r["algorithmVersion"] for r in items}
            strategy = "EMPTY"
            if algorithms:
                strategy = "FALLBACK"
                if algorithms - {"catalog_fallback_v1"}:
                    strategy = {
                        "product_detail": "RELATED",
                        "cart": "BOUGHT_TOGETHER",
                        "home": "PERSONALIZED",
                    }[placement]
            reasons = list(dict.fromkeys(r for r in (reason, ranking_reason) if r))
            if "catalog_fallback_v1" in algorithms and len(algorithms) > 1:
                reasons.append("PARTIAL_FALLBACK")
            decision = uuid4()
            conn.execute(
                """INSERT INTO decision_trace
                   (recommendation_id,scope_id,placement,generation_id,strategy,items)
                   VALUES(%s,%s,%s,%s,%s,%s)""",
                (
                    decision,
                    self.config.scope,
                    placement,
                    generation["generation_id"] if generation else None,
                    strategy,
                    Jsonb([r["skuCode"] for r in items]),
                ),
            )
            return {
                "recommendationId": str(decision),
                "placement": placement,
                "scopeId": self.config.scope,
                "generationId": str(generation["generation_id"]) if generation else None,
                "algorithmVersion": engine.ALGORITHM_VERSION,
                "strategy": strategy,
                "reasonCodes": reasons,
                "asOf": min(r["source_as_of"] for r in sources.values()).isoformat(),
                "items": items,
            }

    def sales(self, conn, now):
        # Supply tables may not be present in an isolated Recommendation deployment.
        available = conn.execute(
            "SELECT to_regclass('analytics_config.supply_export') AS t"
        ).fetchone()
        if available["t"] is None:
            return {}
        end = now.astimezone(engine.TZ).date()
        rows = conn.execute(
            """SELECT DISTINCT ON(sku_code,kind,business_date) sku_code,kind,business_date,
               payload,source_as_of FROM analytics_config.supply_export
               WHERE warehouse_id=%s AND scope_key=%s AND pipeline=%s
               AND business_date >= %s AND business_date < %s
               AND kind IN ('order_day','inventory_day')
               ORDER BY sku_code,kind,business_date,revision DESC""",
            (
                self.config.supply_warehouse,
                self.config.supply_scope,
                self.config.supply_pipeline,
                end - timedelta(days=14),
                end,
            ),
        ).fetchall()
        grouped = defaultdict_rows(rows)
        return {
            sku: sum(
                r["payload"]["soldUnits"] - r["payload"]["returnedUnits"]
                for r in records
                if r["kind"] == "order_day"
            )
            for sku, records in grouped.items()
            if len(records) == 28
            and all(r["source_as_of"] >= now - timedelta(hours=48) for r in records)
            and all(
                r["payload"]["stockoutMinutes"] == 0
                for r in records
                if r["kind"] == "inventory_day"
            )
        }

    def cleanup(self, now):
        with self.connect() as conn:
            self.lock_privacy(conn)
            erasures = conn.execute("SELECT * FROM erasure FOR UPDATE").fetchall()
            for row in erasures:
                subject = row["subject_key"]
                conn.execute(
                    "DELETE FROM behavior_event WHERE subject_key=%s AND consent_version < %s",
                    (subject, row["through_version"]),
                )
                conn.execute(
                    """UPDATE source_snapshot s SET records=(
                       SELECT coalesce(jsonb_agg(CASE WHEN x->>'subjectKey'=%s
                           AND (x->>'consentVersion')::bigint < %s
                           THEN x - 'subjectKey' - 'consentVersion' ELSE x END),'[]'::jsonb)
                       FROM jsonb_array_elements(s.records) x)
                       WHERE dataset='completed_baskets' AND records @> %s::jsonb""",
                    (subject, row["through_version"], Jsonb([{"subjectKey": subject}])),
                )
                # Strip only the affected subject while preserving other customers' purchase signals.
                conn.execute("DELETE FROM erasure WHERE subject_key=%s", (subject,))
            conn.execute(
                "DELETE FROM behavior_event WHERE effective_at < %s", (now - timedelta(days=30),)
            )
            conn.execute(
                "DELETE FROM consent_request WHERE created_at < %s", (now - timedelta(days=1),)
            )
            conn.execute(
                "DELETE FROM decision_trace WHERE decided_at < %s", (now - timedelta(days=7),)
            )
            conn.execute(
                """DELETE FROM generation g WHERE built_at < %s AND NOT EXISTS
                   (SELECT 1 FROM serving_pointer p WHERE p.generation_id=g.generation_id)""",
                (now - timedelta(days=1),),
            )
            conn.execute(
                """DELETE FROM source_snapshot s WHERE received_at < %s AND NOT EXISTS
                   (SELECT 1 FROM source_pointer p WHERE p.snapshot_id=s.snapshot_id)""",
                (now - timedelta(days=1),),
            )


def defaultdict_rows(rows):
    result = {}
    for row in rows:
        result.setdefault(row["sku_code"], []).append(row)
    return result
