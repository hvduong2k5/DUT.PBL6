"""Validate the exact fixture consumed by Go against the canonical producer schema."""
import json
from pathlib import Path
from jsonschema import Draft7Validator, FormatChecker

root = Path(__file__).resolve().parents[1]
schema = json.loads((root / "packages/events/schemas/procurement/v1/goods_received.event.json").read_text(encoding="utf-8"))
fixture = json.loads((root / "services/inventory-service/internal/presentation/kafka/testdata/goods_received.json").read_text(encoding="utf-8"))
Draft7Validator.check_schema(schema)
Draft7Validator(schema, format_checker=FormatChecker()).validate(fixture)
assert fixture["data"]["supplier_id"] == "SUP-HTX-QUANG-THO"
assert fixture["id"] == "receipt-event-01"
assert "quantity_accepted" not in fixture["data"]
print("PASS: canonical GoodsReceived schema validates the Go consumer fixture")
