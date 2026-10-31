"""Validate emitted synthetic Payment examples against the unchanged candidates.

Usage: python scripts/verify-payment-evidence.py --test-log path/to/go-test.jsonl
Requires PyYAML and jsonschema. Does not contact providers or load credentials.
"""
import argparse
import json
from pathlib import Path
import re
import jsonschema
import yaml


def nullable(node):
    if isinstance(node, list):
        return [nullable(value) for value in node]
    if not isinstance(node, dict):
        return node
    result = {key: nullable(value) for key, value in node.items() if key != "nullable"}
    if node.get("nullable"):
        return {"anyOf": [result, {"type": "null"}]}
    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--test-log", type=Path, required=True)
    args = parser.parse_args()
    specs = Path(__file__).resolve().parents[3] / "docs" / "03_api_specs"
    customer = nullable(yaml.safe_load((specs / "order-payment-sepay.candidate.openapi.yaml").read_text(encoding="utf-8")))
    staff = nullable(yaml.safe_load((specs / "order-payment-staff.candidate.openapi.yaml").read_text(encoding="utf-8")))
    names = {"PaymentStatus": (customer, "PaymentStatus"), "PaymentInstructions": (customer, "PaymentInstructions"),
             "Refund": (staff, "Refund"), "RefundStart": (staff, "StartInstruction"), "ReconciliationCase": (staff, "Case")}
    found = set()
    count = 0
    event_count = 0
    for line in args.test_log.read_text(encoding="utf-8-sig").splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        match = re.search(r"PAYMENT_CONTRACT (\w+) (\{.*\})", event.get("Output", ""))
        financial = re.search(r"PAYMENT_EVENT (\{.*\})", event.get("Output", ""))
        if financial:
            payload = json.loads(financial.group(1))
            if payload["type"].startswith("vn.omama.payment."):
                schema = json.loads((specs / "order-payment-events.candidate.schema.json").read_text(encoding="utf-8"))
            else:
                schema = json.loads((specs.parents[1] / "packages/events/schemas/order/v3/paid.event.json").read_text(encoding="utf-8"))
            jsonschema.Draft7Validator(schema, format_checker=jsonschema.FormatChecker()).validate(payload)
            event_count += 1
        if not match:
            continue
        name, payload = match.groups()
        document, schema = names[name]
        root = dict(document)
        root["$ref"] = "#/components/schemas/" + schema
        jsonschema.Draft7Validator(root, format_checker=jsonschema.FormatChecker()).validate(json.loads(payload))
        found.add(name)
        count += 1
    missing = set(names) - found
    if missing:
        raise SystemExit("Missing runtime examples: " + ", ".join(sorted(missing)))
    if event_count < 2:
        raise SystemExit("Missing runtime receipt/paid event examples")
    print(json.dumps({"validated_examples": count, "validated_events": event_count, "schemas": sorted(found)}))


if __name__ == "__main__":
    main()
