import hashlib
import json
import re
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from uuid import UUID
from zoneinfo import ZoneInfo

from jsonschema import Draft7Validator, FormatChecker

TZ = ZoneInfo("Asia/Ho_Chi_Minh")
ORDER_SOURCE = "https://omama.vn/services/order-service"
TYPES = {
    "vn.omama.order.paid.v3": "PAID",
    "vn.omama.order.completed.v3": "COMPLETED",
}
IGNORED = {
    "vn.omama.order.placed.v3",
    "vn.omama.order.cancelled.v3",
    "vn.omama.order.fulfillment_ready.v3",
    "vn.omama.order.reconciliation_required.v3",
    "vn.omama.order.refund_succeeded.v3",
    "vn.omama.order.audit.v3",
}
FRACTION = re.compile(r"\.(\d+)(?:Z|[+-]\d\d:\d\d)$")


def digest(data: bytes) -> bytes:
    return hashlib.sha256(data).digest()


def canonical(value) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


@dataclass(frozen=True)
class Milestone:
    source: str
    event_id: str
    event_type: str
    order_id: UUID
    milestone: str
    occurred_at: datetime
    submicro_ns: int
    order_version: int
    ordinal: int
    normalized_digest: bytes

    @property
    def day(self):
        return self.occurred_at.astimezone(TZ).date()


class InvalidRecord(ValueError):
    pass


class OrderAdapter:
    def __init__(self, schema_dir: Path):
        self.validators = {
            event_type: Draft7Validator(
                json.loads(
                    (schema_dir / f"{milestone.lower()}.event.json").read_text(encoding="utf-8")
                ),
                format_checker=FormatChecker(),
            )
            for event_type, milestone in TYPES.items()
        }

    def decode(self, key: bytes | None, body: bytes) -> Milestone | None:
        if not body or len(body) > 1_048_576:
            raise InvalidRecord("INVALID_ENVELOPE")
        try:
            event = json.loads(body)
        except (ValueError, RecursionError) as exc:
            raise InvalidRecord("INVALID_JSON") from exc
        if not isinstance(event, dict) or event.get("source") != ORDER_SOURCE:
            raise InvalidRecord("INVALID_SOURCE")
        event_type = event.get("type")
        if not isinstance(event_type, str):
            raise InvalidRecord("UNKNOWN_EVENT_TYPE")
        if event_type in IGNORED:
            return None
        validator = self.validators.get(event_type)
        if validator is None:
            raise InvalidRecord("UNKNOWN_EVENT_TYPE")
        if not validator.is_valid(event):
            raise InvalidRecord("SCHEMA_VIOLATION")
        data = event["data"]
        order = data["aggregate_id"]
        if key != order.encode() or event["subject"] != f"order:{order}":
            raise InvalidRecord("ROUTING_MISMATCH")
        if data["version"] >= 2**63 or data["ordinal"] >= 2**31:
            raise InvalidRecord("REVISION_OUT_OF_RANGE")
        timestamp = event["time"].upper()
        try:
            occurred_at = datetime.fromisoformat(timestamp).astimezone(UTC)
            occurred_at.astimezone(TZ)
        except (ValueError, OverflowError) as exc:
            raise InvalidRecord("INVALID_TIMESTAMP") from exc
        fraction = FRACTION.search(timestamp)
        submicro = int((fraction[1] + "000000000")[6:9]) if fraction else 0
        return Milestone(
            event["source"],
            str(UUID(event["id"])),
            event_type,
            UUID(order),
            TYPES[event_type],
            occurred_at,
            submicro,
            data["version"],
            data["ordinal"],
            digest(canonical([event_type, data, occurred_at.isoformat(), submicro])),
        )
