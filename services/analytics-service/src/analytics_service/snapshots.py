import hashlib
from collections import Counter
from datetime import UTC, datetime, time, timedelta
from uuid import UUID, uuid4

from psycopg.types.json import Jsonb

from . import db
from .config import COHORT, TOPIC
from .events import TZ, canonical, digest
from .metrics import BUILD_SECONDS, COVERAGE, PUBLISHED, QUARANTINE
from .migrate import clickhouse

FACT = "analytics_db.order_milestone_fact"
DAILY = "analytics_db.order_milestone_daily"
ORDER_METRICS = {"orders_paid": "PAID", "orders_completed": "COMPLETED"}


def capture(conn, settings, today):
    conn.execute("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY")
    start = today - timedelta(days=179)
    end = today + timedelta(days=1)
    cursors = db.cursors(conn, settings.pipeline)
    quarantine = db.open_quarantine(conn, settings.pipeline)
    QUARANTINE.set(quarantine)
    source_invalid = bool(quarantine)
    gap = not cursors or any(row["gap_code"] for row in cursors)
    proof = (
        conn.execute(
            """SELECT business_date,source_as_of,approved_at,approval_id,paid_count,completed_count
           FROM coverage_approval
           WHERE pipeline=%s AND invalidated_at IS NULL
             AND business_date >= %s AND business_date < %s ORDER BY business_date""",
            (settings.pipeline, start, end),
        ).fetchall()
        if not (source_invalid or gap)
        else []
    )
    coverage = {
        "days": {
            row["business_date"].isoformat(): {
                "sourceAsOf": row["source_as_of"].isoformat(),
                "approvedAt": row["approved_at"].isoformat(),
                "approvalId": row["approval_id"],
                "counts": {"PAID": row["paid_count"], "COMPLETED": row["completed_count"]},
            }
            for row in proof
        },
        "sourceInvalid": source_invalid,
        "gap": gap,
    }
    cutoffs = [
        [row["topic_identity"], row["partition_id"], row["next_offset"], row["gap_code"]]
        for row in cursors
    ]
    input_digest = digest(canonical([start.isoformat(), cutoffs, coverage]))
    current = conn.execute(
        """SELECT p.generation_id,g.input_digest,g.pipeline
           FROM serving_pointer AS p JOIN serving_generation AS g USING (generation_id)
           WHERE p.scope_key=%s""",
        (settings.scope,),
    ).fetchone()
    if current and current["pipeline"] != settings.pipeline:
        raise db.Conflict("Serving scope already belongs to another pipeline")
    if current and current["input_digest"] == input_digest:
        return None
    rows = conn.execute(
        """SELECT cohort,order_id,milestone,occurred_at,event_id FROM order_milestone
           WHERE pipeline=%s AND occurred_at >= %s AND occurred_at < %s
           ORDER BY cohort, order_id, milestone LIMIT %s""",
        (
            settings.pipeline,
            datetime.combine(start, time.min, TZ),
            datetime.combine(end, time.min, TZ),
            settings.max_fact_rows + 1,
        ),
    ).fetchall()
    if len(rows) > settings.max_fact_rows:
        raise db.Conflict("Snapshot exceeds configured fact row budget")
    return (
        current["generation_id"] if current else None,
        input_digest,
        cutoffs,
        coverage,
        rows,
        max((row["source_as_of"] for row in proof), default=None),
    )


def fact_checksum(rows):
    hasher = hashlib.sha256()
    for cohort, order, milestone, occurred, day, event_id in rows:
        if occurred.tzinfo is None:
            occurred = occurred.replace(tzinfo=UTC)
        fields = [
            cohort,
            str(order),
            milestone,
            occurred.astimezone(UTC).isoformat(),
            day.isoformat(),
            event_id,
        ]
        hasher.update(canonical(fields) + b"\n")
    return hasher.digest()


def materialize(client, generation, rows):
    facts = [
        (
            row["cohort"],
            row["order_id"],
            row["milestone"],
            row["occurred_at"],
            row["occurred_at"].astimezone(TZ).date(),
            row["event_id"],
        )
        for row in rows
    ]
    daily = Counter((cohort, day, milestone) for cohort, _, milestone, _, day, _ in facts)
    fact_columns = [
        "cohort",
        "order_id",
        "milestone",
        "occurred_at",
        "business_date",
        "source_event_id",
    ]
    for start in range(0, len(facts), 10_000):
        client.insert(
            FACT,
            [(generation, *row) for row in facts[start : start + 10_000]],
            column_names=["generation_id", *fact_columns],
            settings={"async_insert": 0},
        )
    if daily:
        client.insert(
            DAILY,
            [(generation, *key, count) for key, count in daily.items()],
            column_names=["generation_id", "cohort", "business_date", "milestone", "order_count"],
            settings={"async_insert": 0},
        )
    params = {"generation": generation}
    fact_result = client.query(
        f"""SELECT {", ".join(fact_columns)} FROM {FACT}
           WHERE generation_id={{generation:UUID}} ORDER BY cohort,order_id,toString(milestone)""",
        parameters=params,
        settings={"max_result_rows": len(facts) + 1},
    ).result_rows
    # Compare in PostgreSQL's UUID/text order independently of ClickHouse's UUID/enum ordering.
    ordered_facts = sorted(fact_result, key=lambda row: (row[0], str(row[1]), row[2]))
    checksum = fact_checksum(facts)
    daily_result = client.query(
        f"""SELECT cohort,business_date,milestone,order_count FROM {DAILY}
           WHERE generation_id={{generation:UUID}}""",
        parameters=params,
        settings={"max_result_rows": len(daily) + 1},
    ).result_rows
    if (
        len(ordered_facts) != len(facts)
        or fact_checksum(ordered_facts) != checksum
        or len(daily_result) != len(daily)
        or {tuple(row[:3]): row[3] for row in daily_result} != daily
    ):
        raise db.Conflict("ClickHouse fact or rollup parity failed")
    return len(facts), checksum


def publish(conn, settings, generation, expected, cutoffs, coverage, as_of, count, checksum):
    conn.execute("SELECT pg_advisory_xact_lock(hashtextextended(%s, 12))", (settings.scope,))
    pointer = conn.execute(
        "SELECT generation_id FROM serving_pointer WHERE scope_key=%s FOR UPDATE",
        (settings.scope,),
    ).fetchone()
    if (pointer["generation_id"] if pointer else None) != expected:
        return False
    changed = conn.execute(
        """UPDATE serving_generation SET state='PUBLISHED',published_at=clock_timestamp(),
           fact_count=%s,fact_digest=%s WHERE generation_id=%s AND state='BUILDING'""",
        (count, checksum, generation),
    ).rowcount
    if changed != 1:
        return False
    for identity, partition, offset, _ in cutoffs:
        conn.execute(
            """INSERT INTO generation_cutoff
               (generation_id,topic_identity,topic,partition_id,next_offset) VALUES (%s,%s,%s,%s,%s)""",
            (generation, identity, TOPIC, partition, offset),
        )
    for metric in ORDER_METRICS:
        conn.execute(
            """INSERT INTO metric_definition
               (metric_id,definition_version,unit,owner,policy,effective_from)
               VALUES (%s,'1','ORDER','order-service','{}',clock_timestamp())
               ON CONFLICT DO NOTHING""",
            (metric,),
        )
        conn.execute(
            """INSERT INTO generation_metric
               (generation_id,metric_id,definition_version,source_as_of,coverage)
               VALUES (%s,%s,'1',%s,%s)""",
            (generation, metric, as_of, Jsonb(coverage)),
        )
    conn.execute(
        """INSERT INTO serving_pointer (scope_key,generation_id,updated_at)
           VALUES (%s,%s,clock_timestamp()) ON CONFLICT (scope_key)
           DO UPDATE SET generation_id=EXCLUDED.generation_id,updated_at=EXCLUDED.updated_at""",
        (settings.scope, generation),
    )
    if expected:
        conn.execute(
            """UPDATE serving_generation SET serve_until=clock_timestamp()+interval '24 hours'
               WHERE generation_id=%s""",
            (expected,),
        )
    return True


def abandon(settings, generation):
    with db.connect(settings) as conn:
        conn.execute(
            """UPDATE serving_generation SET state='ABANDONED'
               WHERE generation_id=%s AND state='BUILDING'""",
            (generation,),
        )


def purge(settings, client):
    with db.connect(settings) as conn:
        conn.execute("SELECT pg_advisory_xact_lock(hashtextextended(%s, 12))", (settings.scope,))
        conn.execute(
            """UPDATE serving_generation SET state='ABANDONED'
               WHERE scope_key=%s AND state='BUILDING'
                 AND created_at<clock_timestamp()-interval '30 minutes'""",
            (settings.scope,),
        )
        conn.execute(
            """UPDATE serving_generation AS g SET state='RETIRED'
               WHERE scope_key=%s AND state='PUBLISHED'
                 AND serve_until<clock_timestamp()-interval '1 minute'
                 AND NOT EXISTS (SELECT 1 FROM serving_pointer WHERE generation_id=g.generation_id)""",
            (settings.scope,),
        )
        stale = conn.execute(
            """SELECT generation_id FROM serving_generation AS g WHERE scope_key=%s
               AND state IN ('RETIRED','ABANDONED') AND purged_at IS NULL
               AND created_at<clock_timestamp()-interval '1 minute' ORDER BY created_at LIMIT 5""",
            (settings.scope,),
        ).fetchall()
    for row in stale:
        generation = row["generation_id"]
        for table in (DAILY, FACT):
            client.command(
                f"ALTER TABLE {table} DELETE WHERE generation_id=toUUID('{generation}') "
                "SETTINGS mutations_sync=1",
            )
        with db.connect(settings) as conn:
            conn.execute(
                "UPDATE serving_generation SET purged_at=clock_timestamp() WHERE generation_id=%s",
                (generation,),
            )


@BUILD_SECONDS.time()
def build(settings, client):
    with db.connect(settings) as conn:
        captured = capture(conn, settings, datetime.now(TZ).date())
    if captured is None:
        return None
    expected, input_digest, cutoffs, coverage, rows, as_of = captured
    generation = uuid4()
    with db.connect(settings) as conn:
        conn.execute(
            """INSERT INTO serving_generation
               (generation_id,scope_key,pipeline,expected_generation_id,state,created_at,input_digest)
               VALUES (%s,%s,%s,%s,'BUILDING',clock_timestamp(),%s)""",
            (generation, settings.scope, settings.pipeline, expected, input_digest),
        )
    try:
        count, checksum = materialize(client, generation, rows)
        with db.connect(settings) as conn:
            published = publish(
                conn,
                settings,
                generation,
                expected,
                cutoffs,
                coverage,
                as_of,
                count,
                checksum,
            )
        if published:
            COVERAGE.set(len(coverage["days"]))
            PUBLISHED.set_to_current_time()
            return generation
    except Exception:
        # An unknown SQL commit is reconciled by the conditional update: a published generation stays.
        abandon(settings, generation)
        raise
    abandon(settings, generation)
    return None


def build_once(settings):
    client = clickhouse(settings)
    try:
        generation = build(settings, client)
        purge(settings, client)
        return generation
    finally:
        client.close()


def rollback(settings, generation: UUID):
    with db.connect(settings) as conn:
        conn.execute("SELECT pg_advisory_xact_lock(hashtextextended(%s, 12))", (settings.scope,))
        if not db.get_generation(conn, settings.scope, settings.pipeline, generation):
            raise db.Conflict("Rollback snapshot is unavailable in this scope")
        pointer = conn.execute(
            "SELECT generation_id FROM serving_pointer WHERE scope_key=%s FOR UPDATE",
            (settings.scope,),
        ).fetchone()
        if pointer["generation_id"] == generation:
            return
        conn.execute(
            "UPDATE serving_pointer SET generation_id=%s,updated_at=clock_timestamp() WHERE scope_key=%s",
            (generation, settings.scope),
        )
        conn.execute(
            "UPDATE serving_generation SET serve_until=NULL WHERE generation_id=%s",
            (generation,),
        )
        conn.execute(
            """UPDATE serving_generation SET serve_until=clock_timestamp()+interval '24 hours'
               WHERE generation_id=%s""",
            (pointer["generation_id"],),
        )


def read_daily(client, generation, start, end):
    rows = client.query(
        f"""SELECT business_date,milestone,order_count FROM {DAILY}
           WHERE generation_id={{generation:UUID}} AND cohort={{cohort:String}}
             AND business_date >= {{start:Date}} AND business_date < {{end:Date}}""",
        parameters={"generation": generation, "cohort": COHORT, "start": start, "end": end},
        settings={"max_execution_time": 2, "max_result_rows": 400},
    ).result_rows
    result = {(day, milestone): count for day, milestone, count in rows}
    if len(result) != len(rows):
        raise db.Conflict("Duplicate rollup grain")
    return result
