from __future__ import annotations

import json
import logging
import uuid

logger = logging.getLogger(__name__)


class NullRunRecorder:
    def start(self, conversation_id: str) -> str:
        return str(uuid.uuid4())

    def record_tool(self, record: dict) -> None:
        return None

    def finish(
        self, run_id: str, status: str, steps: int, tool_calls: int, error: str | None
    ) -> None:
        return None


class PostgresRunRecorder:
    def __init__(self, database_url: str):
        self.database_url = database_url

    def _conn(self):
        from ..db import connect

        return connect(self.database_url)

    def start(self, conversation_id: str) -> str:
        run_id = str(uuid.uuid4())
        try:
            with self._conn() as conn:
                conn.execute(
                    "INSERT INTO agent_runs (id, conversation_id) VALUES (%s, %s)",
                    (run_id, conversation_id),
                )
        except Exception:
            logger.exception("failed to persist agent run start")
        return run_id

    def record_tool(self, record: dict) -> None:
        try:
            with self._conn() as conn:
                conn.execute(
                    "INSERT INTO tool_calls "
                    "(run_id, tool_name, args, status, latency_ms) "
                    "VALUES (%s, %s, %s, %s, %s)",
                    (
                        record.get("run_id"),
                        record.get("tool_name"),
                        json.dumps(record.get("args") or {}),
                        record.get("status", "error"),
                        record.get("latency_ms"),
                    ),
                )
        except Exception:
            logger.exception("failed to persist tool call audit")

    def finish(
        self, run_id: str, status: str, steps: int, tool_calls: int, error: str | None
    ) -> None:
        try:
            with self._conn() as conn:
                conn.execute(
                    "UPDATE agent_runs SET status = %s, steps = %s, tool_calls = %s, "
                    "error = %s, finished_at = now() WHERE id = %s",
                    (status, steps, tool_calls, error, run_id),
                )
        except Exception:
            logger.exception("failed to persist agent run completion")
