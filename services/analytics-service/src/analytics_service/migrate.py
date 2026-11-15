import hashlib
import logging

import clickhouse_connect
from clickhouse_connect.driver.exceptions import ClickHouseError
from psycopg import sql

from . import db
from .config import setting

log = logging.getLogger(__name__)


def clickhouse(settings):
    if not settings.clickhouse_host:
        raise ClickHouseError("ClickHouse is not configured")
    return clickhouse_connect.get_client(
        host=settings.clickhouse_host,
        port=settings.clickhouse_port,
        username=settings.clickhouse_user,
        password=settings.clickhouse_password,
        secure=settings.clickhouse_secure,
        connect_timeout=2,
        send_receive_timeout=5,
        query_retries=0,
        autogenerate_session_id=False,
    )


def migrate(settings):
    with db.connect(settings) as conn:
        conn.execute("SELECT pg_advisory_xact_lock(145183602)")
        conn.execute(
            """CREATE TABLE IF NOT EXISTS public.analytics_schema_migration (
               store text NOT NULL, version text NOT NULL, checksum text NOT NULL,
               PRIMARY KEY (store, version))"""
        )
        client = clickhouse(settings) if settings.clickhouse_host else None
        try:
            for store in ("postgres", "clickhouse"):
                if store == "clickhouse" and client is None:
                    log.info("Skipping ClickHouse migrations: ANALYTICS_CLICKHOUSE_HOST is empty")
                    continue
                for path in sorted((settings.migrations / store).glob("*.sql")):
                    body = path.read_text(encoding="utf-8")
                    checksum = hashlib.sha256(body.encode()).hexdigest()
                    key = (store, path.name)
                    existing = conn.execute(
                        """SELECT checksum FROM public.analytics_schema_migration
                           WHERE store=%s AND version=%s""",
                        key,
                    ).fetchone()
                    if existing:
                        if existing["checksum"] != checksum:
                            raise ValueError(f"Migration checksum changed: {path.name}")
                        continue
                    if store == "postgres":
                        conn.execute(body)
                    else:
                        for command in body.split(";"):
                            if command.strip():
                                client.command(command)
                    conn.execute(
                        """INSERT INTO public.analytics_schema_migration (store,version,checksum)
                           VALUES (%s,%s,%s)""",
                        (*key, checksum),
                    )
            for role, privileges in (
                (setting("POSTGRES_WORKER_ROLE"), sql.SQL("SELECT, INSERT, UPDATE")),
                (setting("POSTGRES_API_ROLE"), sql.SQL("SELECT")),
            ):
                if role:
                    identifier = sql.Identifier(role)
                    conn.execute(
                        sql.SQL("GRANT USAGE ON SCHEMA analytics_config TO {}").format(identifier)
                    )
                    conn.execute(
                        sql.SQL("GRANT {} ON ALL TABLES IN SCHEMA analytics_config TO {}").format(
                            privileges, identifier
                        )
                    )
            for role, worker in (
                (setting("REC_POSTGRES_WORKER_ROLE"), True),
                (setting("REC_POSTGRES_API_ROLE"), False),
            ):
                if not role:
                    continue
                identifier = sql.Identifier(role)
                conn.execute(
                    sql.SQL("GRANT USAGE ON SCHEMA analytics_recommendation TO {}").format(
                        identifier
                    )
                )
                conn.execute(
                    sql.SQL(
                        "GRANT SELECT ON ALL TABLES IN SCHEMA analytics_recommendation TO {}"
                    ).format(identifier)
                )
                tables = (
                    (
                        "source_snapshot",
                        "source_pointer",
                        "generation",
                        "serving_pointer",
                        "behavior_event",
                        "consent_request",
                        "privacy_state",
                        "erasure",
                        "decision_trace",
                    )
                    if worker
                    else (
                        "consent",
                        "consent_request",
                        "privacy_state",
                        "erasure",
                        "behavior_event",
                        "decision_trace",
                    )
                )
                privileges = (
                    sql.SQL("INSERT, UPDATE, DELETE") if worker else sql.SQL("INSERT, UPDATE")
                )
                for table in tables:
                    conn.execute(
                        sql.SQL("GRANT {} ON analytics_recommendation.{} TO {}").format(
                            privileges, sql.Identifier(table), identifier
                        )
                    )
                conn.execute(
                    sql.SQL("GRANT USAGE ON SCHEMA analytics_config TO {}").format(identifier)
                )
                conn.execute(
                    sql.SQL("GRANT SELECT ON analytics_config.supply_export TO {}").format(
                        identifier
                    )
                )
        finally:
            if client is not None:
                client.close()
