"""Local OpenAPI/event/document checks; validates actual sandbox outbox payloads."""
import hashlib
import json
import re
import subprocess
import os
import argparse
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import unquote
from urllib.request import Request, urlopen

import yaml
from jsonschema import Draft7Validator, FormatChecker
from openapi_spec_validator import validate_spec
from openapi_schema_validator import OAS30Validator

ROOT = Path(__file__).resolve().parents[3]
EVIDENCE = ROOT / "docs/04_testing/order-service"
COMPOSE = ["docker", "compose", "-f", str(ROOT / "services/order-service/docker-compose.test.yml")]


def command(args):
    return subprocess.check_output(args, cwd=ROOT, text=True, encoding="utf-8").strip()


def sql(query):
    return command(COMPOSE + ["exec", "-T", "db", "psql", "-U", "order_test", "-d", "order_runtime", "-At", "-v", "ON_ERROR_STOP=1", "-c", query])


def http_json(url, method="GET", body=None, headers=None):
    request = Request(url, method=method, data=None if body is None else json.dumps(body).encode(), headers={"Content-Type": "application/json", **(headers or {})})
    with urlopen(request, timeout=3) as response:
        return json.load(response)


options = argparse.ArgumentParser()
options.add_argument("--fresh-fixture", action="store_true", help="Create and cancel an isolated synthetic checkout even when sandbox Orders already exist")
options = options.parse_args()


result = {"time": datetime.now(timezone.utc).isoformat(), "mode": "local_sandbox", "openapi": []}
for name in ["order-service.openapi.yaml", "order-sandbox-dependencies.openapi.yaml"]:
    path = ROOT / "docs/03_api_specs" / name
    document = yaml.safe_load(path.read_text(encoding="utf-8"))
    validate_spec(document)
    result["openapi"].append({"path": str(path.relative_to(ROOT)), "status": "PASS", "paths": len(document["paths"])})

public = yaml.safe_load((ROOT / "docs/03_api_specs/order-service.openapi.yaml").read_text(encoding="utf-8"))


def expand(schema):
    if isinstance(schema, list):
        return [expand(value) for value in schema]
    if not isinstance(schema, dict):
        return schema
    if "$ref" in schema:
        return expand(public["components"]["schemas"][schema["$ref"].rsplit("/", 1)[1]])
    return {key: expand(value) for key, value in schema.items()}


identity = sql("SELECT id::text||'|'||principal_id::text||'|'||operation_id::text FROM orders WHERE scope LIKE 'customer:%' ORDER BY created_at LIMIT 1").split("|")
created_fixture = options.fresh_fixture or len(identity) != 3
if created_fixture:
    principal = str(uuid.uuid4())
    account = http_json("http://127.0.0.1:18104/test/token", "POST", {"user_id": principal, "role": "CUSTOMER"}, {"X-Internal-Token": os.getenv("ORDER_INTERNAL_TOKEN", "order-sandbox-internal-token-2026")})
    auth = {"Authorization": "Bearer " + account["token"]}
    cart = http_json("http://127.0.0.1:18004/api/v1/cart/items", "POST", {"sku_code": "MX-GION-500G", "quantity": 1, "revision": 0}, auth)
    quote = http_json("http://127.0.0.1:18004/api/v1/checkout/quote", "POST", {"payment_method": "VIETQR", "cart_revision": cart["revision"], "shipping_address_id": account["address_id"]}, auth)
    operation = http_json("http://127.0.0.1:18004/api/v1/checkout", "POST", {"payment_method": "VIETQR", "cart_revision": cart["revision"], "quote_id": quote["quote_id"]}, {**auth, "X-Idempotency-Key": str(uuid.uuid4())})
    deadline = time.monotonic() + 30
    while operation["status"] != "SUCCEEDED" and time.monotonic() < deadline:
        assert operation["status"] not in ("FAILED", "MANUAL_REVIEW"), operation
        time.sleep(.1)
        operation = http_json("http://127.0.0.1:18004/api/v1/checkout-operations/" + operation["operation_id"], headers=auth)
    assert operation["status"] == "SUCCEEDED", operation
    identity = [operation["order_id"], principal, operation["operation_id"]]
token_request = Request("http://127.0.0.1:18104/test/token", data=json.dumps({"user_id": identity[1], "role": "CUSTOMER"}).encode(), headers={"Content-Type": "application/json", "X-Internal-Token": os.getenv("ORDER_INTERNAL_TOKEN", "order-sandbox-internal-token-2026")})
with urlopen(token_request, timeout=3) as response:
    token = json.load(response)["token"]
runtime_contracts = []
for resource, name in [("/api/v1/orders/" + identity[0], "Order"), ("/api/v1/checkout-operations/" + identity[2], "CheckoutOperation"), ("/api/v1/cart", "Cart")]:
    with urlopen(Request("http://127.0.0.1:18004" + resource, headers={"Authorization": "Bearer " + token}), timeout=3) as response:
        body = json.load(response)
    OAS30Validator(expand(public["components"]["schemas"][name])).validate(body)
    runtime_contracts.append({"schema": name, "status": "PASS"})
result["http_runtime_schemas"] = runtime_contracts
if created_fixture:
    current = http_json("http://127.0.0.1:18004/api/v1/orders/" + identity[0], headers={"Authorization": "Bearer " + token})
    http_json("http://127.0.0.1:18004/api/v1/orders/" + identity[0] + "/cancel", "POST", {"reason": "contract fixture cleanup"}, {"Authorization": "Bearer " + token, "If-Match": str(current["version"]), "X-Idempotency-Key": str(uuid.uuid4())})
result["http_fixture"] = {"created": created_fixture, "cancel_requested": created_fixture}

validators = {}
for path in sorted((ROOT / "packages/events/schemas/order/v2").glob("*.json")):
    schema = json.loads(path.read_text(encoding="utf-8"))
    Draft7Validator.check_schema(schema)
    validators[schema["properties"]["type"]["const"]] = Draft7Validator(schema, format_checker=FormatChecker())
payloads = sql("SELECT payload::text FROM outbox ORDER BY created_at,id").splitlines()
types = set()
for raw in payloads:
    payload = json.loads(raw)
    validators[payload["type"]].validate(payload)
    types.add(payload["type"])
assert payloads, "No actual outbox payloads to verify"
result["events"] = {"schema_count": len(validators), "runtime_payloads": len(payloads), "runtime_types": sorted(types), "status": "PASS", "note": "Schema definitions checked for all types; runtime validation covers listed types only."}

folders = [ROOT / "docs/02_architecture/services/order-service"]
docs = [p for folder in folders for p in folder.glob("*.md")]
docs += [EVIDENCE / "01_acceptance_and_verification_plan.md", EVIDENCE / "03_implementation_verification.md", ROOT / "docs/06_operations/order_service_runbook.md", ROOT / "services/order-service/README.md"]
links = 0
for path in docs:
    text = re.sub(r"```.*?```", "", path.read_text(encoding="utf-8"), flags=re.S)
    for match in re.finditer(r"!?\[[^\]]*\]\(([^)]+)\)", text):
        target = match.group(1).strip().strip("<>").split("#", 1)[0]
        if not target or re.match(r"(?:https?://|app://|codex://|mailto:)", target):
            continue
        resolved = (path.parent / unquote(target)).resolve()
        assert resolved.exists(), f"Broken link in {path.relative_to(ROOT)}: {target}"
        links += 1
result["documentation"] = {"files": len(docs), "local_links": links, "status": "PASS"}

before = {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest() for p in (ROOT / "services/order-service/internal/gen").rglob("*.go")}
command(["docker", "run", "--rm", "-v", str(ROOT) + ":/workspace", "omamx/order-go-tools:local", "sh", "-c", 'sh scripts/generate-proto.sh && test -z "$(gofmt -l cmd internal migrations tests)"'])
after = {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest() for p in (ROOT / "services/order-service/internal/gen").rglob("*.go")}
assert before == after, "Generated protobuf drift"
result["protobuf_format"] = {"generated_files": len(after), "status": "PASS"}
yaml.safe_load((ROOT / "services/api-gateway/config/kong.yml").read_text(encoding="utf-8"))
result["kong"] = {"yaml_parse": "PASS", "runtime_proxy": "NOT_RUN"}

EVIDENCE.mkdir(parents=True, exist_ok=True)
output = EVIDENCE / "contracts-verification.json"
# Create before link scanning on the first run via a checked-in initial artifact.
schema_versions = sql("SELECT version::text FROM order_migrations ORDER BY version")
images = json.loads(command(COMPOSE + ["images", "--format", "json"]))
tracked = sorted((ROOT / "services/order-service").rglob("*.go")) + sorted((ROOT / "services/order-service/migrations").glob("*.sql"))
manifest = {"time": result["time"], "git_head_base": command(["git", "rev-parse", "HEAD"]), "worktree": "uncommitted implementation; base commit is not a release commit", "schema_versions": schema_versions.splitlines(), "images": images, "app_mode": "sandbox", "seed": "synthetic catalog/profile fixtures in internal/simulator/schema.sql and test helpers", "config": "docker-compose.test.yml defaults; separate runtime/simulator DBs; loopback ports; real PostgreSQL/Redis/Kafka", "source_sha256": {str(p.relative_to(ROOT)): hashlib.sha256(p.read_bytes()).hexdigest() for p in tracked}}
(EVIDENCE / "environment-evidence.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(json.dumps(result, ensure_ascii=False, indent=2))
