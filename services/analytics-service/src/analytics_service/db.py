from contextlib import contextmanager
from datetime import UTC, date, datetime, time, timedelta
from uuid import UUID

import psycopg
from psycopg.rows import dict_row
from psycopg.types.json import Jsonb

from .config import COHORT, TOPIC
from .events import TZ, Milestone, digest


class Conflict(ValueError):
    pass


@contextmanager
def connect(settings):
    with psycopg.connect(
        settings.postgres_dsn,
        row_factory=dict_row,
        connect_timeout=2,
        options="-c search_path=analytics_config,public -c statement_timeout=5000",
    ) as conn:
        yield conn


def pipeline_lock(conn, pipeline):
    conn.execute("SELECT pg_advisory_xact_lock(hashtextextended(%s, 11))", (pipeline,))


def cursors(conn, pipeline):
    return conn.execute(
        "SELECT * FROM partition_cursor WHERE pipeline=%s AND topic=%s ORDER BY partition_id",
        (pipeline, TOPIC),
    ).fetchall()


def bootstrap(conn, pipeline, identity, offsets):
    pipeline_lock(conn, pipeline)
    existing = cursors(conn, pipeline)
    if existing:
        actual = {row["partition_id"]: row["bootstrap_offset"] for row in existing}
        if actual != offsets or any(row["topic_identity"] != identity for row in existing):
            raise Conflict("Pipeline already bootstrapped with different coordinates")
        return
    with conn.cursor() as cursor:
        cursor.executemany(
            """INSERT INTO partition_cursor
               (pipeline,topic,partition_id,topic_identity,next_offset,bootstrap_offset,updated_at)
               VALUES (%s,%s,%s,%s,%s,%s,clock_timestamp())""",
            [
                (pipeline, TOPIC, partition, identity, offset, offset)
                for partition, offset in offsets.items()
            ],
        )


def claim(conn, pipeline, identity, partition, owner):
    return conn.execute(
        """UPDATE partition_cursor SET lease_owner=%s,
           lease_epoch=lease_epoch + CASE
             WHEN lease_owner=%s AND lease_until>clock_timestamp() THEN 0 ELSE 1 END,
           lease_until=clock_timestamp()+interval '15 seconds'
           WHERE pipeline=%s AND topic=%s AND partition_id=%s AND topic_identity=%s
             AND gap_code IS NULL AND
             (lease_owner=%s OR lease_until IS NULL OR lease_until<=clock_timestamp())
           RETURNING next_offset,lease_epoch,last_record_digest""",
        (owner, owner, pipeline, TOPIC, partition, identity, owner),
    ).fetchone()


def release(conn, pipeline, owner, partitions):
    conn.execute(
        """UPDATE partition_cursor SET lease_owner=NULL,lease_until=NULL
           WHERE pipeline=%s AND topic=%s AND lease_owner=%s AND partition_id=ANY(%s)""",
        (pipeline, TOPIC, owner, list(partitions)),
    )


def mark_gap(conn, pipeline, reason, partition=None):
    pipeline_lock(conn, pipeline)
    conn.execute(
        """UPDATE partition_cursor SET gap_code=%s, lease_owner=NULL, lease_until=NULL
           WHERE pipeline=%s AND topic=%s AND (%s::integer IS NULL OR partition_id=%s)""",
        (reason, pipeline, TOPIC, partition, partition),
    )
    invalidate(conn, pipeline, reason)


def invalidate(conn, pipeline, reason, day=None):
    conn.execute(
        """UPDATE coverage_approval
           SET invalidated_at=clock_timestamp(), invalidation_reason=%s
           WHERE pipeline=%s AND invalidated_at IS NULL
             AND (%s::date IS NULL OR business_date=%s)""",
        (reason, pipeline, day, day),
    )


def _append(conn, pipeline, identity, partition, offset, event: Milestone):
    prior_event = conn.execute(
        """SELECT normalized_digest FROM source_event
           WHERE pipeline=%s AND source=%s AND event_id=%s""",
        (pipeline, event.source, event.event_id),
    ).fetchone()
    if prior_event:
        return (
            None
            if prior_event["normalized_digest"] == event.normalized_digest
            else "EVENT_CONFLICT"
        )
    key = (pipeline, COHORT, event.order_id, event.milestone)
    prior = conn.execute(
        """SELECT occurred_at,submicro_ns,order_version,ordinal FROM order_milestone
           WHERE pipeline=%s AND cohort=%s AND order_id=%s AND milestone=%s""",
        key,
    ).fetchone()
    projection = (event.occurred_at, event.submicro_ns, event.order_version, event.ordinal)
    if prior and tuple(prior.values()) != projection:
        return "MILESTONE_CONFLICT"
    conn.execute(
        """INSERT INTO source_event
           (pipeline,source,event_id,event_type,adapter_version,normalized_digest,occurred_at,
            ingested_at,topic_identity,topic,partition_id,source_offset)
           VALUES (%s,%s,%s,%s,'1',%s,%s,clock_timestamp(),%s,%s,%s,%s)""",
        (
            pipeline,
            event.source,
            event.event_id,
            event.event_type,
            event.normalized_digest,
            event.occurred_at,
            identity,
            TOPIC,
            partition,
            offset,
        ),
    )
    if not prior:
        conn.execute(
            """INSERT INTO order_milestone
               (pipeline,cohort,order_id,milestone,occurred_at,submicro_ns,order_version,ordinal,
                source,event_id) VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)""",
            (*key, *projection, event.source, event.event_id),
        )
        invalidate(conn, pipeline, "NEW_MILESTONE", event.day)
    return None


def ingest(conn, pipeline, identity, partition, offset, owner, epoch, raw, event, error):
    pipeline_lock(conn, pipeline)
    current = conn.execute(
        """SELECT next_offset FROM partition_cursor
           WHERE pipeline=%s AND topic=%s AND partition_id=%s AND topic_identity=%s
             AND lease_owner=%s AND lease_epoch=%s AND lease_until>clock_timestamp()
             AND gap_code IS NULL FOR UPDATE""",
        (pipeline, TOPIC, partition, identity, owner, epoch),
    ).fetchone()
    if current is None:
        raise Conflict("Partition lease expired")
    if offset < current["next_offset"]:
        return current["next_offset"]
    if offset > current["next_offset"]:
        mark_gap(conn, pipeline, "UNEXPECTED_OFFSET", partition)
        return None
    if event is not None:
        error = _append(conn, pipeline, identity, partition, offset, event)
    raw_digest = digest(raw)
    if error:
        conn.execute(
            """INSERT INTO source_quarantine
               (pipeline,topic_identity,topic,partition_id,source_offset,reason_code,
                record_digest,detected_at)
               VALUES (%s,%s,%s,%s,%s,%s,%s,clock_timestamp())""",
            (pipeline, identity, TOPIC, partition, offset, error, raw_digest),
        )
        invalidate(conn, pipeline, "SOURCE_INVALID")
    conn.execute(
        """UPDATE partition_cursor SET next_offset=%s,last_record_digest=%s,updated_at=clock_timestamp()
           WHERE pipeline=%s AND topic=%s AND partition_id=%s""",
        (offset + 1, raw_digest, pipeline, TOPIC, partition),
    )
    return offset + 1


def open_quarantine(conn, pipeline):
    return conn.execute(
        """SELECT count(*) AS n FROM source_quarantine
           WHERE pipeline=%s AND resolved_at IS NULL""",
        (pipeline,),
    ).fetchone()["n"]


def approve_day(
    conn, pipeline, day: date, identity, offsets, paid, completed, as_of, actor, evidence
):
    if (
        as_of.tzinfo is None
        or not actor.strip()
        or not evidence.strip()
        or type(paid) is not int
        or type(completed) is not int
        or min(paid, completed) < 0
    ):
        raise Conflict(
            "Coverage requires valid counts, zoned timestamp, actor and evidence reference"
        )
    start = datetime.combine(day, time.min, TZ)
    end = start + timedelta(days=1)
    if not end <= as_of <= datetime.now(UTC):
        raise Conflict("Coverage can only certify a closed business day")
    pipeline_lock(conn, pipeline)
    rows = cursors(conn, pipeline)
    if not rows or any(row["gap_code"] or row["topic_identity"] != identity for row in rows):
        raise Conflict("Coverage requires the expected topic and all healthy partitions")
    if (
        offsets.keys() != {row["partition_id"] for row in rows}
        or any(
            not row["bootstrap_offset"] <= offsets[row["partition_id"]] <= row["next_offset"]
            for row in rows
        )
        or open_quarantine(conn, pipeline)
    ):
        raise Conflict("Coverage requires reached journal cutoffs and resolved quarantine")
    found = {
        row["milestone"]: (row["total"], row["proven"])
        for row in conn.execute(
            """SELECT m.milestone,count(*) AS total,
                      count(*) FILTER (
                          WHERE s.source_offset < (%s::jsonb ->> s.partition_id::text)::bigint
                      ) AS proven
               FROM order_milestone AS m JOIN source_event AS s USING (pipeline,source,event_id)
               WHERE m.pipeline=%s AND m.cohort=%s AND s.topic_identity=%s
                 AND m.occurred_at >= %s AND m.occurred_at < %s
               GROUP BY m.milestone""",
            (Jsonb(offsets), pipeline, COHORT, identity, start, end),
        )
    }
    if found.get("PAID", (0, 0)) != (paid, paid) or found.get("COMPLETED", (0, 0)) != (
        completed,
        completed,
    ):
        raise Conflict(
            "Coverage counts do not match the journal at the approved and current cutoffs"
        )
    existing = conn.execute(
        """SELECT approval_id,invalidated_at FROM coverage_approval
           WHERE pipeline=%s AND business_date=%s AND topic_identity=%s AND cutoffs=%s
             AND paid_count=%s AND completed_count=%s AND source_as_of=%s AND evidence_ref=%s
           ORDER BY approval_id DESC LIMIT 1""",
        (pipeline, day, identity, Jsonb(offsets), paid, completed, as_of, evidence),
    ).fetchone()
    if existing:
        if existing["invalidated_at"] is not None:
            raise Conflict("Previously invalidated evidence requires a new source confirmation")
        return existing["approval_id"]
    invalidate(conn, pipeline, "SUPERSEDED", day)
    return conn.execute(
        """INSERT INTO coverage_approval
           (pipeline,business_date,topic_identity,cutoffs,paid_count,completed_count,
            source_as_of,approved_by,evidence_ref) VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)
           RETURNING approval_id""",
        (pipeline, day, identity, Jsonb(offsets), paid, completed, as_of, actor, evidence),
    ).fetchone()["approval_id"]


def repair(conn, pipeline, identity, partition, offset, resolution, raw=None, event=None):
    pipeline_lock(conn, pipeline)
    record = conn.execute(
        """SELECT record_digest FROM source_quarantine
           WHERE pipeline=%s AND topic_identity=%s AND partition_id=%s AND source_offset=%s
             AND resolved_at IS NULL FOR UPDATE""",
        (pipeline, identity, partition, offset),
    ).fetchone()
    if record is None:
        raise Conflict("No matching open quarantine record")
    if raw is not None:
        if digest(raw) != record["record_digest"] or event is None:
            raise Conflict("Repair requires the original record and a valid milestone")
        reason = _append(conn, pipeline, identity, partition, offset, event)
        if reason:
            raise Conflict(reason)
    conn.execute(
        """UPDATE source_quarantine SET resolved_at=clock_timestamp(),resolution_ref=%s
           WHERE pipeline=%s AND topic_identity=%s AND partition_id=%s AND source_offset=%s""",
        (resolution, pipeline, identity, partition, offset),
    )


def get_generation(conn, scope, pipeline, generation_id: UUID | None = None):
    return conn.execute(
        """SELECT g.generation_id,g.published_at,m.source_as_of,m.coverage
           FROM serving_generation AS g
           JOIN generation_metric AS m USING (generation_id)
           LEFT JOIN serving_pointer AS p ON p.scope_key=g.scope_key
           WHERE g.scope_key=%s AND g.pipeline=%s AND g.state='PUBLISHED'
             AND m.metric_id='orders_paid'
             AND g.generation_id=COALESCE(%s::uuid,p.generation_id)
             AND (g.serve_until IS NULL OR g.serve_until>clock_timestamp())""",
        (scope, pipeline, generation_id),
    ).fetchone()
