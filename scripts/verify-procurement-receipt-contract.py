"""Validate real Procurement event bytes; optionally compare an unmerged Inventory ref.

No network, Git writes, broker or DB mutations. Requires MVP Python dependencies and Go.
"""
import argparse
import copy
from decimal import Decimal
import json
from pathlib import Path
import subprocess

from jsonschema import Draft7Validator, FormatChecker

ROOT = Path(__file__).resolve().parents[1]
SCHEMA = "packages/events/schemas/procurement/v1/goods_received.event.json"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--inventory-ref", help="Optional local Git revision, e.g. PR #118 head 21775d66")
    args = parser.parse_args()
    schemas = {"current": json.loads((ROOT / SCHEMA).read_text(encoding="utf-8"), parse_float=Decimal)}
    if args.inventory_ref:
        # Resolve to an object ID before using it as a git show argument.
        revision = subprocess.run(["git", "rev-parse", "--verify", "--end-of-options", args.inventory_ref + "^{commit}"],
                                  cwd=ROOT, check=True, capture_output=True, text=True).stdout.strip()
        raw = subprocess.run(["git", "show", revision + ":" + SCHEMA], cwd=ROOT,
                             check=True, capture_output=True, encoding="utf-8").stdout
        schemas["inventory:" + revision] = json.loads(raw, parse_float=Decimal)
    result = subprocess.run(["go", "test", "-p", "1", "./internal/domain", "-run", "^TestGoodsReceivedContractExamples$", "-count=1", "-v"],
                            cwd=ROOT / "services/procurement-service", check=True, capture_output=True, encoding="utf-8")
    events = [json.loads(line.split("CONTRACT_EVENT=", 1)[1], parse_float=Decimal)
              for line in result.stdout.splitlines() if "CONTRACT_EVENT=" in line]
    if len(events) != 3:
        raise AssertionError("Expected real accepted/partial/rejected producer examples")
    for name, schema in schemas.items():
        Draft7Validator.check_schema(schema)
        validator = Draft7Validator(schema, format_checker=FormatChecker())
        for event in events:
            validator.validate(event)
        for unit, quantity in (("BOX", 1), ("KG", Decimal("1.000001")),
                               ("UNIT", Decimal("1.5")), ("G", 2147483648), ("KG", Decimal("0.001"))):
            invalid = copy.deepcopy(events[0])
            invalid["data"].update(unit=unit, quantity_received=quantity)
            if validator.is_valid(invalid):
                raise AssertionError(f"{name} accepted incompatible quantity {unit}/{quantity}")
        invalid = copy.deepcopy(events[0])
        invalid.pop("traceparent")
        if validator.is_valid(invalid):
            raise AssertionError(f"{name} accepted missing trace context")
        print(f"PASS: {name}: 3 real producer examples and 6 invalid cases")
    print("Schema compatibility only; live consumer, QC and deployment E2E NOT RUN")


if __name__ == "__main__":
    main()
