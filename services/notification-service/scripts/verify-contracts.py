"""Validate pinned producer fixture and local command with installed jsonschema."""
import hashlib
import json
from pathlib import Path

from jsonschema import Draft7Validator, Draft202012Validator, FormatChecker

root = Path(__file__).resolve().parents[1] / "contracts"
manifest = json.loads((root / "events/manifest.json").read_text())
assert manifest["enabled"] is False
for event in manifest["events"]:
    raw = (root / "events" / event["schema"]).read_bytes()
    assert hashlib.sha256(raw).hexdigest() == event["sha256"]
    schema = json.loads(raw)
    Draft7Validator.check_schema(schema)
    fixture = json.loads((root / "examples/order.placed.v3.json").read_text())
    Draft7Validator(schema, format_checker=FormatChecker()).validate(fixture)

api = json.loads((root / "notification.openapi.json").read_text())
schema = api["components"]["schemas"]["Command"]
Draft202012Validator.check_schema(schema)
fixture = json.loads((root / "examples/intake.json").read_text())
Draft202012Validator(schema, format_checker=FormatChecker()).validate(fixture)
fixture["purpose"] = "OTP"
assert not Draft202012Validator(schema).is_valid(fixture)
print("PASS pinned producer schema/fixture and local command schema/positive/negative fixtures")
