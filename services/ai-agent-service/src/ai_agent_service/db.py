from __future__ import annotations

import pathlib

import psycopg


def connect(database_url: str) -> psycopg.Connection:
    return psycopg.connect(database_url, autocommit=True)


def ping(database_url: str) -> bool:
    try:
        with connect(database_url) as conn:
            conn.execute("SELECT 1")
        return True
    except Exception:
        return False


def run_migrations(database_url: str, migrations_dir: str) -> list[str]:
    """Apply pending *.sql migrations in name order; returns applied file names."""
    folder = pathlib.Path(migrations_dir)
    files = sorted(folder.glob("*.sql"))
    applied: list[str] = []
    with connect(database_url) as conn:
        conn.execute("SELECT pg_advisory_lock(hashtext('ai-agent-service:migrations'))")
        try:
            conn.execute(
                "CREATE TABLE IF NOT EXISTS schema_migrations "
                "(name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())"
            )
            done = {row[0] for row in conn.execute("SELECT name FROM schema_migrations")}
            for path in files:
                if path.name in done:
                    continue
                conn.execute(path.read_text(encoding="utf-8"))
                conn.execute("INSERT INTO schema_migrations (name) VALUES (%s)", (path.name,))
                applied.append(path.name)
        finally:
            conn.execute("SELECT pg_advisory_unlock(hashtext('ai-agent-service:migrations'))")
    return applied
