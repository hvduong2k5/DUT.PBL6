"""Validate captured local HTTP test responses against the authoritative contracts."""
import argparse
import copy
import json
from functools import cache
from pathlib import Path
import yaml
from jsonschema import Draft202012Validator, FormatChecker

SERVICE = Path(__file__).resolve().parents[1]
SPECS = SERVICE.parents[1] / "docs/03_api_specs"
SCOPED = SPECS / "traceability-scoped-admin-draft.openapi.yaml"
INTEGRATION = SPECS / "traceability-workflow-draft.openapi.yaml"
LEGACY = SPECS / "traceability-service.openapi.yaml"

@cache
def document(path):
    return yaml.safe_load(path.read_text(encoding="utf-8"))

def expand(node, path):
    if isinstance(node, list):
        return [expand(value, path) for value in node]
    if not isinstance(node, dict):
        return node
    node = copy.deepcopy(node)
    if "$ref" in node:
        filename, _, pointer = node.pop("$ref").partition("#")
        target_path = (path.parent / filename).resolve() if filename else path
        target = document(target_path)
        for part in pointer.lstrip("/").split("/"):
            if part:
                target = target[part.replace("~1", "/").replace("~0", "~")]
        return {**expand(target, target_path), **expand(node, path)}
    nullable = node.pop("nullable", False)
    if nullable and "type" in node:
        node["type"] = [node["type"], "null"]
    return {key: expand(value, path) for key, value in node.items()}


CASES = {
    "create": (SCOPED, "/api/v2/admin/trace/records", "post", 201),
    "published": (SCOPED, "/api/v2/admin/trace/records/{id}/revisions/{revision}/publish", "post", 200),
    "list": (SCOPED, "/api/v2/admin/trace/records", "get", 200),
    "unauthenticated": (SCOPED, "/api/v2/admin/trace/records", "get", 401),
    "context": (INTEGRATION, "/api/v2/admin/trace/batches/{batch_id}/context", "get", 200),
    "lookup-empty": (INTEGRATION, "/api/v2/admin/trace/records/by-batch", "get", 200),
    "public": (LEGACY, "/api/v1/trace/{qr_code}", "get", 200),
}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("artifact_dir", type=Path)
    args = parser.parse_args()
    for name, (source, path, method, status) in CASES.items():
        actual = json.loads((args.artifact_dir / (name + ".json")).read_text(encoding="utf-8"))
        assert actual["status"] == status, name
        assert actual["cacheControl"] == "no-store", name
        operation = expand(document(source)["paths"][path], source)[method]
        response = operation["responses"][str(status)]
        schema = response["content"]["application/json"]["schema"]
        Draft202012Validator.check_schema(schema)
        Draft202012Validator(schema, format_checker=FormatChecker()).validate(actual["body"])
        if "requestId" in actual["body"]:
            assert actual["body"]["requestId"] == actual["requestId"], name
    print(f"PASS: {len(CASES)} captured local HTTP responses match authoritative OpenAPI schemas and headers.")


if __name__ == "__main__":
    main()
