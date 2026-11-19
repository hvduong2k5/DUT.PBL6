"""Offline artifact verification; never connects to peers, brokers or databases.

Run from any directory. Optional grpcio-tools/pglast expand static checks.
--report writes a machine-readable result; runtime acceptance remains PENDING.
"""
from __future__ import annotations

import argparse
import copy
import hashlib
import importlib
import importlib.util
import importlib.metadata
import json
import re
import subprocess
import sys
import tempfile
import xml.etree.ElementTree as ET
from pathlib import Path

import yaml
from jsonschema import Draft7Validator, FormatChecker
from jsonschema.exceptions import ValidationError
from openapi_spec_validator import validate_spec


ROOT = Path(__file__).resolve().parents[3]
ARCH = ROOT / "docs/02_architecture/services/fulfillment-service"
API = ROOT / "docs/03_api_specs/fulfillment-service"
SPEC = ROOT / "docs/03_api_specs/fulfillment-service.candidate.openapi.yaml"
CHECKS: list[dict] = []


def record(name: str, status: str = "PASS", **details) -> None:
    CHECKS.append({"name": name, "status": status, **details})


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def load(path: Path):
    return json.loads(path.read_text(encoding="utf-8"))


def oas_jsonschema(value):
    """Translate only the OAS3.0 schema features used in this candidate."""
    if isinstance(value, list):
        return [oas_jsonschema(v) for v in value]
    if not isinstance(value, dict):
        return value
    result = {k: oas_jsonschema(v) for k, v in value.items() if k != "nullable"}
    if value.get("nullable"):
        return {"anyOf": [result, {"type": "null"}]}
    return result


def check_links() -> None:
    groups = [ARCH, ROOT / "docs/02_architecture/adr/fulfillment-service", API,
              ROOT / "docs/04_testing/fulfillment-service"]
    files = [p for directory in groups for p in directory.rglob("*.md")]
    files += [ROOT / "docs/05_deployment/fulfillment_service_deployment.md",
              ROOT / "docs/06_operations/fulfillment_service_runbook.md"]
    count = 0
    for path in files:
        text = path.read_text(encoding="utf-8")
        require(text.count("```") % 2 == 0, f"Unbalanced code fence: {path}")
        for target in re.findall(r"\[[^\]]+\]\(([^)]+)\)", text):
            target = target.strip("<>").split("#", 1)[0]
            if not target or re.match(r"[a-z]+://", target):
                continue
            require((path.parent / target).resolve().exists(), f"Broken link {path}: {target}")
            count += 1
    record("markdown_links_and_fences", files=len(files), local_links=count)


def check_manifest() -> None:
    manifest = load(ARCH / "source_manifest.json")
    verified, unavailable = 0, 0
    for source in manifest["sources"]:
        require(re.fullmatch(r"[0-9a-f]{40}", source["commit"]) is not None, "Unpinned source")
        result = subprocess.run(["git", "show", f'{source["commit"]}:{source["path"]}'],
                                cwd=ROOT, capture_output=True, check=False)
        if result.returncode != 0:
            unavailable += 1  # A source commit can be absent in a future shallow clone.
            continue
        require(hashlib.sha256(result.stdout).hexdigest() == source["sha256"],
                f'Source hash mismatch: {source["path"]}')
        verified += 1
    for snapshot in manifest["snapshots"]:
        data = (ROOT / snapshot["local_path"]).read_bytes()
        require(hashlib.sha256(data).hexdigest() == snapshot["sha256"], "Baseline snapshot modified")
    record("pinned_source_manifest", verified_blobs=verified, unavailable_blobs=unavailable,
           snapshot_count=len(manifest["snapshots"]))
    if unavailable:
        record("source_blobs_not_in_local_git", "SKIP", count=unavailable)


def check_openapi():
    spec = yaml.safe_load(SPEC.read_text(encoding="utf-8"))
    validate_spec(spec)
    converted = oas_jsonschema(spec)
    components = converted["components"]["schemas"]

    def validate(name, example):
        schema = {**components[name], "components": converted["components"]}
        Draft7Validator(schema, format_checker=FormatChecker()).validate(example)

    operations = []
    for path, item in spec["paths"].items():
        for method, operation in item.items():
            if method not in {"get", "post", "put", "patch", "delete"}:
                continue
            operations.append(operation["operationId"])
            require(operation.get("x-permissions"), f"Missing permission: {path}")
            parameter_names = {p.get("$ref", "").rsplit("/", 1)[-1] for p in operation["parameters"]}
            if "IdempotencyKey" in parameter_names:
                require(operation.get("x-command-action"), f"Missing command action mapping: {path}")
            if method in {"post", "put"} and "/tasks/{task_id}" in path and not path.endswith("media-sessions"):
                params = {p.get("$ref", "").rsplit("/", 1)[-1] for p in operation["parameters"]}
                require({"IdempotencyKey", "IfMatch"} <= params, f"Missing guards: {path}")
    require(len(operations) == len(set(operations)), "Duplicate operationId")
    component_examples = 0
    for name, schema in spec["components"]["schemas"].items():
        if "example" in schema:
            validate(name, schema["example"])
            component_examples += 1
    examples = load(API / "examples.json")
    for key, name in {"task": "Task", "task_page": "TaskPage", "seal_request": "SealRequest",
                      "seal_receipt": "SealReceipt", "media_file": "MediaFile",
                      "error": "Error", "command_outcome": "CommandOutcome"}.items():
        validate(name, examples[key])
    record("openapi_and_positive_examples", operations=len(operations), schemas=len(components),
           component_examples=component_examples, linked_examples=8)

    negatives = []
    def invalid(name, payload, label):
        try:
            validate(name, payload)
        except ValidationError:
            negatives.append(label)
            return
        raise AssertionError(f"Invalid sample accepted: {label}")

    bad = copy.deepcopy(examples["seal_request"]); bad["parcel"]["weight_grams"] = 0
    invalid("SealRequest", bad, "zero_weight")
    bad = copy.deepcopy(examples["seal_request"]); bad["parcel"]["length_cm"] = 1.5
    invalid("SealRequest", bad, "fractional_dimensions")
    bad = copy.deepcopy(examples["seal_request"]); bad["role"] = "ADMIN"
    invalid("SealRequest", bad, "caller_role_injection")
    bad = copy.deepcopy(examples["seal_request"]); bad["seal_code"] = "lowercase-seal"
    invalid("SealRequest", bad, "unnormalized_seal")
    bad = copy.deepcopy(examples["task"]); bad["version"] = 0
    invalid("Task", bad, "zero_task_version")
    bad = copy.deepcopy(examples["task"]); bad["items"][0]["quantity"] = -1
    invalid("Task", bad, "negative_quantity")
    invalid("AssignRequest", {"user_role": "ADMIN"}, "assignment_role_injection")
    invalid("ChecklistRequest", {"result": "PASS", "all_checked": True}, "checklist_bypass")
    invalid("MediaSessionRequest", {"package_id": examples["seal_request"]["package_id"],
                                   "media_type": "application/x-executable", "size_bytes": 100,
                                   "sha256": "a" * 64}, "unsupported_media_type")
    record("openapi_negative_guards", rejected=len(negatives), cases=negatives)
    return examples


def event_runtime_shape(event) -> None:
    """Additional shape checks visible in pinned Shipping consumer; no authority claim."""
    data = event["data"]
    require(event["source"] == "https://omama.vn/services/fulfillment-service", "Invalid source")
    require(event["subject"] == "order_id:" + data["order_id"], "Invalid subject binding")
    require(re.fullmatch(r"00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}", event["traceparent"]) is not None,
            "Invalid traceparent")
    require(bool(data["s3_video_object_key"].strip()), "Empty video key")
    require(bool(data["packer_staff_id"].strip()), "Empty staff reference")


def check_event_and_bindings(examples) -> None:
    schema = load(API / "packing_completed.v2.baseline.schema.json")
    Draft7Validator.check_schema(schema)
    validator = Draft7Validator(schema, format_checker=FormatChecker())
    event = examples["packing_completed_v2"]
    validator.validate(event)
    event_runtime_shape(event)
    rejected = 0
    cases = [("missing_package", lambda e: e["data"].pop("package_id")),
             ("missing_video", lambda e: e["data"].pop("s3_video_object_key")),
             ("bad_checksum", lambda e: e["data"].update(video_checksum_sha256="no")),
             ("bad_generation", lambda e: e["data"].update(ready_generation=0)),
             ("bad_task_uuid", lambda e: e["data"].update(task_id="task-not-uuid")),
             ("empty_video", lambda e: e["data"].update(s3_video_object_key="")),
             ("forged_source", lambda e: e.update(source="https://forged.invalid/source")),
             ("wrong_subject", lambda e: e.update(subject="order:" + e["data"]["order_id"]))]
    for label, mutate in cases:
        candidate = copy.deepcopy(event)
        mutate(candidate)
        try:
            validator.validate(candidate)
            event_runtime_shape(candidate)
        except (ValidationError, AssertionError):
            rejected += 1
        else:
            raise AssertionError(f"Invalid event accepted: {label}")
    snapshot = examples["packing_snapshot"]
    allocation = examples["committed_allocation"]
    validation = examples["packing_validation"]
    proof = examples["package_revision"]
    task = examples["task"]
    data = event["data"]
    for name in ["order_id", "ready_generation", "items_digest", "warehouse_id"]:
        values = [x[name] for x in [snapshot, allocation, validation, proof, task]]
        require(len(set(values)) == 1, f"Example binding mismatch: {name}")
    totals = {}
    for line in snapshot["lines"]:
        totals[line["sku_code"]] = totals.get(line["sku_code"], 0) + line["quantity"]
    items = [{"quantity": totals[sku], "sku_code": sku} for sku in sorted(totals)]
    canonical = json.dumps(items, ensure_ascii=True, separators=(",", ":"), sort_keys=True)
    require(hashlib.sha256(canonical.encode()).hexdigest() == snapshot["items_digest"], "Digest example mismatch")
    require(data["task_id"] == task["task_id"] == proof["task_id"], "Task binding mismatch")
    require(data["package_id"] == proof["package_id"] == examples["seal_receipt"]["package_id"], "Package binding mismatch")
    require(data["resource_version"] == proof["sealed_task_version"] == examples["seal_receipt"]["version"],
            "Sealed task version mismatch")
    require(data["parcel"] == proof["parcel"] == examples["seal_request"]["parcel"], "Parcel mismatch")
    require(data["packed_at"] == proof["packed_at"] == examples["seal_receipt"]["packed_at"], "Original time mismatch")
    require(data["video_checksum_sha256"] == examples["media_file"]["verified_sha256"], "Media checksum mismatch")
    for line in snapshot["lines"]:
        slots = [s for s in allocation["slots"] if s["order_line_id"] == line["order_line_id"]]
        require(sum(s["allocated_quantity"] for s in slots) == line["quantity"], "Allocation coverage mismatch")
    record("packing_event_and_cross_artifact_bindings", rejected_negative_events=rejected,
           digest=snapshot["items_digest"], note="Synthetic shape/binding verification only; no real media or peer proof.")


def check_hash_vectors() -> None:
    vectors = load(API / "canonical_hash_vectors.json")["vectors"]
    for vector in vectors:
        canonical = json.dumps(vector["canonical_model"], ensure_ascii=False, sort_keys=True,
                               separators=(",", ":"))
        require(canonical == vector["canonical_utf8"], f'Canonical bytes differ: {vector["name"]}')
        require(hashlib.sha256(canonical.encode("utf-8")).hexdigest() == vector["sha256"],
                f'Golden hash differs: {vector["name"]}')
    record("canonical_hash_vectors", vectors=len(vectors),
           note="Candidate interop vectors; independent Go/Java provider acceptance remains pending.")


def check_accepted_candidate(examples) -> None:
    schema = load(API / "task_accepted.v1.candidate.schema.json")
    Draft7Validator.check_schema(schema)
    event = examples["task_accepted_v1"]
    Draft7Validator(schema, format_checker=FormatChecker()).validate(event)
    require(event["subject"] == "order_id:" + event["data"]["order_id"], "Accepted subject mismatch")
    for field in ["order_id", "task_id", "ready_generation", "items_digest", "warehouse_id"]:
        require(event["data"][field] == examples["task"][field], "Accepted binding mismatch: " + field)
    require(event["data"]["resource_version"] == 1, "Admission checkpoint must be version1")
    require(event["data"]["occurred_at"] == examples["task"]["admitted_at"], "Accepted original time mismatch")
    record("task_accepted_candidate_schema", note="New discovery fact proposal; no shared runtime acceptance.")


def check_proto(examples) -> None:
    try:
        from grpc_tools import protoc
        import grpc_tools
        from google.protobuf import json_format
    except ImportError:
        record("candidate_proto_compile_and_examples", "SKIP", reason="grpcio-tools not available")
        return
    with tempfile.TemporaryDirectory(prefix="fulfillment-proto-") as directory:
        builtin = Path(grpc_tools.__file__).parent / "_proto"
        source = "fulfillment-dependencies.candidate.proto"
        rc = protoc.main(["protoc", f"-I{API}", f"-I{builtin}", f"--python_out={directory}",
                          f"--descriptor_set_out={directory}/candidate.pb", source])
        require(rc == 0, "Candidate proto compile failed")
        generated = list(Path(directory).rglob("*_pb2.py"))
        require(len(generated) == 1, "Missing generated candidate module")
        module_spec = importlib.util.spec_from_file_location("fulfillment_candidate_pb2", generated[0])
        module = importlib.util.module_from_spec(module_spec)
        module_spec.loader.exec_module(module)
        mappings = {"packing_snapshot": "PackingSnapshot", "committed_allocation": "CommittedPackingAllocation",
                    "packing_validation": "PackingValidation", "package_revision": "PackageRevision"}
        for key, name in mappings.items():
            json_format.ParseDict(examples[key], getattr(module, name)(), ignore_unknown_fields=False)
        record("candidate_proto_compile_and_examples", services=len(module.DESCRIPTOR.services_by_name),
               messages=len(module.DESCRIPTOR.message_types_by_name), examples=len(mappings))


def check_sql_and_diagrams() -> None:
    sql = (ARCH / "fulfillment_persistence.candidate.sql").read_text(encoding="utf-8")
    try:
        from pglast import parse_sql
    except ImportError:
        record("candidate_sql_parse", "SKIP", reason="pglast not available; no database was contacted")
    else:
        statements = parse_sql(sql)
        record("candidate_sql_parse", statements=len(statements), note="Syntax only; no DB constraints/races executed.")
    diagram = ET.parse(ARCH / "fulfillment_service_architecture_all_in_one.drawio")
    pages = diagram.getroot().findall("diagram")
    require(len(pages) == 4, "Expected four Draw.io pages")
    for page in pages:
        cells = page.findall(".//mxCell")
        ids = [cell.attrib["id"] for cell in cells]
        require(len(ids) == len(set(ids)), "Duplicate Draw.io cell IDs")
        for cell in cells:
            for attr in ["source", "target"]:
                if attr in cell.attrib:
                    require(cell.attrib[attr] in ids, "Invalid diagram edge")
    record("drawio_structure", pages=len(pages), note="XML/edge validation; not a visual render test.")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", type=Path, help="Write JSON report to the specified file")
    args = parser.parse_args()
    failure = None
    try:
        check_links()
        check_manifest()
        examples = check_openapi()
        check_event_and_bindings(examples)
        check_hash_vectors()
        check_accepted_candidate(examples)
        check_proto(examples)
        check_sql_and_diagrams()
    except Exception as error:
        failure = str(error)
        record("verification_failure", "FAIL", error=failure)
    versions = {}
    for package in ["PyYAML", "jsonschema", "openapi-spec-validator", "grpcio-tools", "protobuf", "pglast"]:
        try:
            versions[package] = importlib.metadata.version(package)
        except importlib.metadata.PackageNotFoundError:
            versions[package] = "not-installed"
    result = {"design_version": "1.0", "python_version": sys.version.split()[0],
              "validation_dependencies": versions, "checks": CHECKS,
              "result": "FAIL" if failure else "PASS", "runtime_acceptance": "PENDING",
              "scope": "Static documents/contracts/examples only; no peer/service/database calls."}
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(result, ensure_ascii=True, indent=2))
    return 1 if failure else 0


if __name__ == "__main__":
    raise SystemExit(main())
