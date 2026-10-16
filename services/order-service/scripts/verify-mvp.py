"""Validate MVP wire contracts and actual synthetic integration-test responses.

Run after the Docker Go gates. Never calls SePay or moves money.
"""
import copy
import json
from pathlib import Path

import yaml
from jsonschema import Draft7Validator, FormatChecker
from openapi_spec_validator import validate_spec
from openapi_schema_validator import OAS30Validator

ROOT = Path(__file__).resolve().parents[3]


def expand(doc, value):
    if isinstance(value, list):
        return [expand(doc, item) for item in value]
    if not isinstance(value, dict):
        return value
    if "$ref" in value:
        return expand(doc, doc["components"]["schemas"][value["$ref"].rsplit("/", 1)[1]])
    return {k: expand(doc, v) for k, v in value.items()}


def main():
    document = yaml.safe_load((ROOT / "docs/03_api_specs/order-service.mvp1.openapi.yaml").read_text(encoding="utf-8"))
    validate_spec(document)
    evidence = ROOT / "docs/04_testing/order-service/mvp1-payloads"
    checked = []
    for name, schema_name in [("checkout", "SePayCheckout"), ("manual-refund", "ManualRefund"), ("payment-review", "PaymentReview")]:
        value = json.loads((evidence / f"{name}.json").read_text(encoding="utf-8"))
        OAS30Validator(expand(document, document["components"]["schemas"][schema_name]), format_checker=FormatChecker()).validate(value)
        checked.append(name)

    schema = json.loads((ROOT / "packages/events/schemas/order/manual/v1/manual_refund_verified.event.json").read_text(encoding="utf-8"))
    Draft7Validator.check_schema(schema)
    validator = Draft7Validator(schema, format_checker=FormatChecker())
    event = json.loads((evidence / "manual-refund-event.json").read_text(encoding="utf-8"))
    validator.validate(event)
    example = json.loads((ROOT / "packages/events/examples/order/manual/v1/manual_refund_verified.json").read_text(encoding="utf-8"))
    validator.validate(example)
    assert event["subject"] == "refund:" + event["data"]["refund_id"]
    rejected = 0
    for key, value in [("amount", 0), ("currency", "USD"), ("execution_mode", "SEPAY_API"), ("evidence_hash", ""), ("status", "SUBMITTED"), ("bank_account", "must-never-be-in-event")]:
        invalid = copy.deepcopy(event)
        invalid["data"][key] = value
        assert list(validator.iter_errors(invalid)), key
        rejected += 1

    result = {"scope": "Order first delivery, not all product MVP 1.0", "openapi": "PASS", "paths": len(document["paths"]), "actual_synthetic_responses": checked, "manual_event": "PASS", "invalid_events_rejected": rejected, "sepay_merchant_sandbox": "NOT_RUN_BY_THIS_CHECK; see live-sepay-sandbox-verification.json", "production": "DISABLED", "dependency_adapters": "SIMULATOR"}
    path = ROOT / "docs/04_testing/order-service/mvp1-contracts-verification.json"
    path.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
