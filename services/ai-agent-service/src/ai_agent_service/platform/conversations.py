from __future__ import annotations

import uuid
from dataclasses import dataclass, field
from datetime import UTC, datetime, timedelta
from typing import Protocol


@dataclass
class Conversation:
    id: str
    owner_type: str  # guest | customer
    customer_id: str | None
    expires_at: datetime


@dataclass
class StoredMessage:
    role: str
    content: str
    metadata: dict = field(default_factory=dict)


class ConversationStore(Protocol):
    def create(
        self, owner_type: str, customer_id: str | None, retention_days: int
    ) -> Conversation: ...

    def get(self, conversation_id: str) -> Conversation | None: ...

    def add_message(
        self, conversation_id: str, role: str, content: str, metadata: dict | None = None
    ) -> None: ...

    def list_messages(self, conversation_id: str) -> list[StoredMessage]: ...

    def get_slots(self, conversation_id: str) -> dict: ...

    def set_slots(self, conversation_id: str, slots: dict) -> None: ...

    def get_history_consent(self, customer_id: str) -> bool: ...

    def set_history_consent(self, customer_id: str, allowed: bool) -> None: ...

    def cleanup_expired(self, now: datetime) -> int: ...


def _now() -> datetime:
    return datetime.now(UTC)


class MemoryConversationStore:
    """In-memory store for tests and fixture-mode development."""

    def __init__(self):
        self._conversations: dict[str, Conversation] = {}
        self._messages: dict[str, list[StoredMessage]] = {}
        self._slots: dict[str, dict] = {}
        self._consent: dict[str, bool] = {}

    def create(self, owner_type, customer_id, retention_days):
        conv = Conversation(
            id=uuid.uuid4().hex,
            owner_type=owner_type,
            customer_id=customer_id,
            expires_at=_now() + timedelta(days=retention_days),
        )
        self._conversations[conv.id] = conv
        self._messages[conv.id] = []
        self._slots[conv.id] = {}
        return conv

    def get(self, conversation_id):
        return self._conversations.get(conversation_id)

    def add_message(self, conversation_id, role, content, metadata=None):
        self._messages.setdefault(conversation_id, []).append(
            StoredMessage(role=role, content=content, metadata=metadata or {})
        )

    def list_messages(self, conversation_id):
        return list(self._messages.get(conversation_id, []))

    def get_slots(self, conversation_id):
        return dict(self._slots.get(conversation_id, {}))

    def set_slots(self, conversation_id, slots):
        self._slots[conversation_id] = dict(slots)

    def get_history_consent(self, customer_id):
        return self._consent.get(customer_id, False)

    def set_history_consent(self, customer_id, allowed):
        self._consent[customer_id] = allowed

    def cleanup_expired(self, now):
        expired = [cid for cid, c in self._conversations.items() if c.expires_at <= now]
        for cid in expired:
            self._conversations.pop(cid, None)
            self._messages.pop(cid, None)
            self._slots.pop(cid, None)
        return len(expired)


class PostgresConversationStore:
    """PostgreSQL-backed store (production). Sync psycopg calls; the queries
    are short and run outside the token-streaming section of a turn."""

    def __init__(self, database_url: str):
        self.database_url = database_url

    def _conn(self):
        from ..db import connect

        return connect(self.database_url)

    def create(self, owner_type, customer_id, retention_days):
        expires = _now() + timedelta(days=retention_days)
        with self._conn() as conn:
            row = conn.execute(
                "INSERT INTO conversations (owner_type, customer_id, expires_at) "
                "VALUES (%s, %s, %s) RETURNING id",
                (owner_type, customer_id, expires),
            ).fetchone()
            conv_id = str(row[0])
            conn.execute(
                "INSERT INTO conversation_contexts (conversation_id) VALUES (%s)", (conv_id,)
            )
        return Conversation(conv_id, owner_type, customer_id, expires)

    def get(self, conversation_id):
        with self._conn() as conn:
            row = conn.execute(
                "SELECT id, owner_type, customer_id, expires_at FROM conversations WHERE id = %s",
                (conversation_id,),
            ).fetchone()
        if row is None:
            return None
        return Conversation(str(row[0]), row[1], row[2], row[3])

    def add_message(self, conversation_id, role, content, metadata=None):
        import json

        with self._conn() as conn:
            conn.execute(
                "INSERT INTO messages (conversation_id, role, content, metadata) "
                "VALUES (%s, %s, %s, %s)",
                (conversation_id, role, content, json.dumps(metadata or {})),
            )
            conn.execute(
                "UPDATE conversations SET updated_at = now() WHERE id = %s", (conversation_id,)
            )

    def list_messages(self, conversation_id):
        with self._conn() as conn:
            rows = conn.execute(
                "SELECT role, content, metadata FROM messages "
                "WHERE conversation_id = %s ORDER BY created_at, id",
                (conversation_id,),
            ).fetchall()
        return [StoredMessage(role=r[0], content=r[1], metadata=r[2] or {}) for r in rows]

    def get_slots(self, conversation_id):
        with self._conn() as conn:
            row = conn.execute(
                "SELECT slots FROM conversation_contexts WHERE conversation_id = %s",
                (conversation_id,),
            ).fetchone()
        return dict(row[0]) if row and row[0] else {}

    def set_slots(self, conversation_id, slots):
        import json

        with self._conn() as conn:
            conn.execute(
                "INSERT INTO conversation_contexts (conversation_id, slots) VALUES (%s, %s) "
                "ON CONFLICT (conversation_id) DO UPDATE SET slots = EXCLUDED.slots, "
                "updated_at = now()",
                (conversation_id, json.dumps(slots)),
            )

    def get_history_consent(self, customer_id):
        with self._conn() as conn:
            row = conn.execute(
                "SELECT history_use FROM consent_records WHERE customer_id = %s", (customer_id,)
            ).fetchone()
        return bool(row[0]) if row else False

    def set_history_consent(self, customer_id, allowed):
        with self._conn() as conn:
            conn.execute(
                "INSERT INTO consent_records (customer_id, history_use) VALUES (%s, %s) "
                "ON CONFLICT (customer_id) DO UPDATE SET history_use = EXCLUDED.history_use, "
                "version = consent_records.version + 1, updated_at = now()",
                (customer_id, allowed),
            )

    def cleanup_expired(self, now):
        with self._conn() as conn:
            rows = conn.execute(
                "DELETE FROM conversations WHERE expires_at <= %s RETURNING id", (now,)
            ).fetchall()
        return len(rows)
