"""Static contract checks only; never calls APIs, Kafka, or provider mutations."""
import argparse
import copy
import json
import re
import subprocess
from pathlib import Path

import yaml
from jsonschema import Draft7Validator, FormatChecker
from openapi_spec_validator import validate_spec
from openapi_schema_validator import OAS30Validator

ROOT = Path(__file__).resolve().parents[2]


def check(condition, message):
    if not condition:
        raise AssertionError(message)


def load_json(path):
    return json.loads(path.read_text(encoding="utf-8"))


def expand(document, value):
    if isinstance(value, list):
        return [expand(document, item) for item in value]
    if not isinstance(value, dict):
        return value
    if "$ref" in value:
        check(value["$ref"].startswith("#/components/schemas/"), "Nonlocal schema ref")
        return expand(document, document["components"]["schemas"][value["$ref"].rsplit("/", 1)[1]])
    return {key: expand(document, item) for key, item in value.items()}


def protobuf_checks(raw, baseline=None):
    from google.protobuf.descriptor_pb2 import FileDescriptorSet
    current = FileDescriptorSet.FromString(raw)
    files = {f.name: f for f in current.file}
    messages, enums, services = {}, {}, {}

    def index(descriptor):
        ms, es, ss = {}, {}, {}
        def walk(message, prefix):
            name = prefix + message.name
            ms[name] = message
            for enum in message.enum_type:
                es[name + "." + enum.name] = enum
            for nested in message.nested_type:
                walk(nested, name + ".")
        for file in descriptor.file:
            for message in file.message_type:
                walk(message, file.package + ".")
            for enum in file.enum_type:
                es[file.package + "." + enum.name] = enum
            for service in file.service:
                ss[file.package + "." + service.name] = service
        return ms, es, ss

    messages, enums, services = index(current)
    for name, fields in {
        "omamx.inventory.v1.ReservationOutcome": ["command", "items_digest", "resource_version", "reversal_id", "cancellation_tombstone"],
        "omamx.inventory.v1.ReservationCommand": ["request_hash", "approval", "fulfillment_stopped", "query_action"],
        "omamx.fulfillment.v1.TaskDecision": ["command", "decision", "generation", "items_digest"],
        "omamx.common.v1.FulfillmentPermit": ["audience", "task_request_key", "expires_at"],
    }.items():
        check(name in messages, f"Missing message {name}")
        names = {field.name for field in messages[name].field}
        check(set(fields) <= names, f"Missing proof fields {name}")
    for name, methods in {
        "omamx.inventory.v1.InventoryService": ["GetReservationByOrder", "FinalizeReservation", "ReleaseReservationByOrder", "ReverseCommittedStock"],
        "omamx.fulfillment.v1.FulfillmentService": ["AcceptTask", "CancelTask", "GetTaskDecision"],
        "omamx.order.v1.OrderService": ["GetFulfillmentPermit", "ValidateFulfillmentPermit"],
        "omamx.catalog.v1.CatalogService": ["GetCheckoutSnapshot"],
        "omamx.promotion.v1.PromotionService": ["GetVoucherHoldByOrder", "FinalizeVoucherHold", "ReleaseVoucherHoldByOrder"],
    }.items():
        check(name in services, f"Missing service {name}")
        check(set(methods) <= {m.name for m in services[name].method}, f"Missing methods {name}")
    checked_fields = 0
    if baseline:
        old = FileDescriptorSet.FromString(baseline)
        old_messages, old_enums, old_services = index(old)
        for file in old.file:
            check(file.name in files, f"Removed file {file.name}")
            now = files[file.name]
            check(file.package == now.package and file.options == now.options, f"Changed package/options: {file.name}")
        for name, message in old_messages.items():
            check(name in messages, f"Removed message {name}")
            by_tag = {field.number: field for field in messages[name].field}
            for field in message.field:
                check(field.number in by_tag, f"Removed tag {name}.{field.number}")
                new_field = by_tag[field.number]
                for attr in ["name", "json_name", "type", "type_name", "label", "proto3_optional"]:
                    check(getattr(field, attr) == getattr(new_field, attr), f"Changed {attr}: {name}.{field.name}")
                # Optional fields use synthetic oneofs; compare names, not positions.
                old_oneof = message.oneof_decl[field.oneof_index].name if field.HasField("oneof_index") else None
                new_message = messages[name]
                new_oneof = new_message.oneof_decl[new_field.oneof_index].name if new_field.HasField("oneof_index") else None
                check(old_oneof == new_oneof, f"Changed oneof {name}.{field.name}")
                checked_fields += 1
        for name, enum in old_enums.items():
            check(name in enums, f"Removed enum {name}")
            new_values = {v.name: v.number for v in enums[name].value}
            for value in enum.value:
                check(new_values.get(value.name) == value.number, f"Changed enum value {name}.{value.name}")
        for name, service in old_services.items():
            check(name in services, f"Removed service {name}")
            methods = {m.name: m for m in services[name].method}
            for method in service.method:
                check(method.name in methods, f"Removed RPC {name}.{method.name}")
                check(method == methods[method.name], f"Changed RPC signature {name}.{method.name}")
    return {"status": "PASS", "files_including_imports": len(files), "baseline_fields_checked": checked_fields,
            "compatibility_scope": "existing file/package/options/message fields/enums/RPC signatures; semantic compatibility requires cohort gates"}


def main():
    args = argparse.ArgumentParser()
    args.add_argument("--compile-proto", action="store_true", help="Compile all shared protos in pinned local Docker tools image")
    args.add_argument("--descriptor", type=Path)
    args.add_argument("--baseline-descriptor", type=Path)
    args.add_argument("--output", type=Path)
    options = args.parse_args()
    check(not options.baseline_descriptor or options.compile_proto or options.descriptor, "Baseline requires current descriptor")
    result = {"mode": "static_design_contracts", "provider_integration": "NOT_RUN", "openapi": [], "events": {}}
    target = None
    for name in ["order-service.openapi.yaml", "order-service.sandbox-3.1.openapi.yaml", "order-service.mvp1.openapi.yaml", "order-sandbox-dependencies.openapi.yaml"]:
        document = yaml.safe_load((ROOT / "docs/03_api_specs" / name).read_text(encoding="utf-8"))
        validate_spec(document)
        result["openapi"].append({"file": name, "status": "PASS", "paths": len(document["paths"])})
        if name == "order-service.openapi.yaml":
            target = document
    check(target["info"]["version"] == "3.2.0" and target["x-contract-status"] == "proposed", "Target version/status drift")
    key_schema = next(p["schema"] for p in target["paths"]["/api/v1/checkout"]["post"]["parameters"] if p["name"] == "X-Idempotency-Key")
    check(re.fullmatch(key_schema["pattern"], "checkout-key-123456"), "Idempotency key rejects valid ASCII input")
    check(not re.fullmatch(key_schema["pattern"], "checkout key 123456"), "Idempotency key accepts whitespace")
    schemas = target["components"]["schemas"]
    examples = 0
    for name, schema in schemas.items():
        if "example" in schema:
            OAS30Validator(expand(target, schema)).validate(schema["example"])
            examples += 1
    operation_validator = OAS30Validator(expand(target, schemas["CheckoutOperation"]))
    accepted = schemas["CheckoutOperation"]["example"]
    succeeded = {**accepted, "status": "SUCCEEDED", "stage": "DONE", "order_id": "11111111-1111-4111-8111-111111111111"}
    operation_validator.validate(succeeded)
    check(not operation_validator.is_valid({**succeeded, "order_id": None}), "SUCCEEDED must identify Order")
    check(not operation_validator.is_valid({**accepted, "order_id": succeeded["order_id"]}), "Acceptance must not expose planned Order")
    result["openapi_examples"] = {"valid": examples + 1, "invalid_rejected": 2}

    validators, payloads = {}, {}
    for path in sorted((ROOT / "packages/events/schemas/order/v3").glob("*.json")):
        schema = load_json(path)
        Draft7Validator.check_schema(schema)
        validator = Draft7Validator(schema, format_checker=FormatChecker())
        name = path.name.removesuffix(".event.json")
        payload = load_json(ROOT / "packages/events/examples/order/v3" / (name + ".json"))
        validator.validate(payload)
        check(payload["subject"] == "order:" + payload["data"]["aggregate_id"], f"Subject mismatch {name}")
        validators[name], payloads[name] = validator, payload
    check(len(validators) == 8, "Expected 8 v3 event types")
    negative = 0
    def reject(name, mutate):
        nonlocal negative
        bad = copy.deepcopy(payloads[name])
        mutate(bad)
        check(not validators[name].is_valid(bad), f"Invalid {name} accepted: {bad}")
        negative += 1
    for name in validators:
        reject(name, lambda p: p["data"].update(phone="0900000000"))
        reject(name, lambda p: p["data"].pop("ordinal"))
        reject(name, lambda p: p.update(type=p["type"].replace(".v3", ".v2")))
    reject("paid", lambda p: p["data"].update(stock_status="RESERVED"))
    reject("paid", lambda p: p["data"].pop("stock_decision_id"))
    reject("paid", lambda p: p["data"].update(payment_status="UNPAID"))
    reject("fulfillment_ready", lambda p: p["data"].update(ready_generation=0))
    reject("fulfillment_ready", lambda p: p["data"].update(stock_status="EXPIRED"))
    reject("fulfillment_ready", lambda p: p["data"].update(status="CONFIRMED_COD", payment_status="CONFIRMED"))
    reject("refund_succeeded", lambda p: p["data"].update(amount=0))
    reject("refund_succeeded", lambda p: p["data"].update(currency="USD"))
    cod = copy.deepcopy(payloads["fulfillment_ready"])
    cod["data"].update(status="CONFIRMED_COD", payment_status="UNPAID")
    validators["fulfillment_ready"].validate(cod)
    result["events"] = {"status": "PASS", "schemas": len(validators), "valid_examples": len(payloads) + 1, "invalid_rejected": negative}
    if options.compile_proto:
        names = sorted(str(path.relative_to(ROOT / "packages/proto")).replace("\\", "/") for path in (ROOT / "packages/proto").rglob("*.proto"))
        compiled = subprocess.run(["docker", "run", "--rm", "-v", str(ROOT) + ":/workspace", "omamx/order-go-tools:local", "protoc", "-I", "/workspace/packages/proto", "--include_imports", "--descriptor_set_out=/dev/stdout", *names], check=True, stdout=subprocess.PIPE)
        raw = compiled.stdout
    elif options.descriptor:
        raw = options.descriptor.read_bytes()
    else:
        raw = None
    if raw:
        result["protobuf"] = protobuf_checks(raw, options.baseline_descriptor.read_bytes() if options.baseline_descriptor else None)
    else:
        result["protobuf"] = {"status": "NOT_RUN", "reason": "Pass --compile-proto or --descriptor"}
    output = json.dumps(result, ensure_ascii=False, indent=2) + "\n"
    if options.output:
        options.output.parent.mkdir(parents=True, exist_ok=True)
        options.output.write_text(output, encoding="utf-8")
    print(output)


if __name__ == "__main__":
    main()
