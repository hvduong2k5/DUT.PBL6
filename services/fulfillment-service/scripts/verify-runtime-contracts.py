"""Validate actual Go HTTP/event payloads; does not call external providers."""
import copy
import json
import hashlib
from pathlib import Path
import yaml
from jsonschema import Draft7Validator, FormatChecker
from openapi_spec_validator import validate_spec

ROOT = Path(__file__).resolve().parents[3]
SERVICE = ROOT / "services/fulfillment-service"
manifest=json.loads((SERVICE/"contracts/SOURCE.json").read_text(encoding="utf-8"))
for name,expected in manifest["files"].items():
    snapshot=(SERVICE/"contracts"/name).read_bytes().replace(b"\r\n",b"\n")
    canonical=ROOT/"packages/proto"/("packing/v1/packing.proto" if name=="packing.proto" else name)
    assert hashlib.sha256(snapshot).hexdigest()==expected, "Pinned snapshot hash mismatch: "+name
    assert canonical.read_bytes().replace(b"\r\n",b"\n")==snapshot, "Canonical contract drift: "+name
def translate(v):
    if isinstance(v,list):return [translate(x) for x in v]
    if not isinstance(v,dict):return v
    out={k:translate(x) for k,x in v.items() if k!="nullable"}
    return {"anyOf":[out,{"type":"null"}]} if v.get("nullable") else out
spec=yaml.safe_load((ROOT/"docs/03_api_specs/fulfillment-service.candidate.openapi.yaml").read_text(encoding="utf-8"))
validate_spec(spec)
components=translate(spec["components"])
task=json.loads((SERVICE/"tests/results/task.json").read_text(encoding="utf-8"))
schema={**components["schemas"]["Task"],"components":components}
Draft7Validator(schema,format_checker=FormatChecker()).validate(task)
for file,relative in [("accepted-event.json","docs/03_api_specs/fulfillment-service/task_accepted.v1.candidate.schema.json"),("packing-event.json","docs/03_api_specs/fulfillment-service/packing_completed.v2.baseline.schema.json"),("audit-event.json","packages/events/schemas/audit/v1/audit_record.event.json")]:
    schema=json.loads((ROOT/relative).read_text(encoding="utf-8"))
    event=json.loads((SERVICE/"tests/results"/file).read_text(encoding="utf-8"))
    Draft7Validator(schema,format_checker=FormatChecker()).validate(event)
report={"result":"PASS","http_payloads":1,"event_payloads":3,"scope":"Actual Go test payloads; media metadata uses a synthetic verification fixture; no owning-peer E2E acceptance."}
(SERVICE/"tests/results/contracts.json").write_text(json.dumps(report,indent=2)+"\n")
print(json.dumps(report,indent=2))
