"""Check real Procurement HTTP payloads against the shared spec and Kong allowlist.

Runs Go HTTP/JWT tests, not a live Identity/Kong deployment. Requires the MVP
Python dependencies and Go. Kong 3.9.3 parsing is a separate container check.
"""
import argparse
import json
from pathlib import Path
import re
import subprocess
import time
from urllib.parse import urlsplit
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

from jsonschema import Draft202012Validator, FormatChecker, ValidationError
import yaml

ROOT = Path(__file__).resolve().parents[1]
PREFIX = "/api/v1/admin/procurement/"


def resolve(spec, value):
    while "$ref" in value:
        ref = value["$ref"]
        assert ref.startswith("#/"), "Procurement schemas must stay in the shared contract"
        value = spec
        for part in ref[2:].split("/"):
            value = value[part.replace("~1", "/").replace("~0", "~")]
    return value


def validate(spec, schema, body):
    Draft202012Validator({**schema, "components": spec["components"]},
                         format_checker=FormatChecker()).validate(body)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--gateway-url", help="Optional disposable Kong debug proxy, loopback HTTP only; no live upstreams")
    args = parser.parse_args()
    spec = yaml.safe_load((ROOT / "docs/03_api_specs/admin-stable.openapi.yaml").read_text(encoding="utf-8"))
    kong = yaml.safe_load((ROOT / "services/api-gateway/config/kong.yml").read_text(encoding="utf-8"))
    services = [s for s in kong["services"] if s["name"] == "procurement-service"]
    assert len(services) == 1
    service = services[0]
    assert service["url"] == "http://procurement-service:8008"
    assert service["retries"] == 0, "Do not silently add transport replay"
    assert len(service["routes"]) == 1
    route = service["routes"][0]
    assert route["strip_path"] is False
    assert set(route["methods"]) == {"GET", "OPTIONS"}
    assert "plugins" not in service and "plugins" not in route, "Do not inject user/internal credentials"
    patterns = []
    for path in route["paths"]:
        assert path.startswith("~/") and path.endswith("$")
        assert "(?" not in path, "Kong expression regex does not support look-around"
        patterns.append(re.compile(path[1:]))

    def matches(path, method):
        return method in route["methods"] and any(p.fullmatch(path) for p in patterns)

    operations = {path: item["get"] for path, item in spec["paths"].items() if path.startswith(PREFIX)}
    assert len(operations) == 6, "Keep this first gateway surface read-only and explicit"
    for path, operation in operations.items():
        assert operation["security"] == [{"bearerAuth": []}], "No implicit cookie authentication"
        assert operation["x-required-grants"] == ["PROCUREMENT_VIEW#GLOBAL"]
        concrete = re.sub(r"\{[^}]+\}", "10000000-0000-0000-0000-000000000001", path)
        assert matches(concrete, "GET") and matches(concrete, "OPTIONS")
        for method in ("POST", "PUT", "PATCH", "DELETE", "HEAD"):
            assert not matches(concrete, method), (concrete, method)
        for response in operation["responses"].values():
            media = resolve(spec, response)["content"]["application/json"]
            examples = [e["value"] for e in media.get("examples", {}).values()]
            if "example" in media:
                examples.append(media["example"])
            assert examples, "Each documented state needs an example"
            for example in examples:
                validate(spec, media["schema"], example)

    for path in ("/health/live", "/health/ready", PREFIX + "purchase-orders",
                 PREFIX + "goods-receipts", PREFIX + "supply-plans", PREFIX + "contracts/x",
                 PREFIX + "suppliers/x/contracts", PREFIX + "suppliers/x/status",
                 PREFIX + "materials/x/preferred-supplier", PREFIX + "suppliers/x/offers/extra",
                 PREFIX + "suppliers/x/offers/", PREFIX + "suppliers-extra", PREFIX + "suppliers/"):
        assert not matches(path, "GET"), path

    compose = yaml.safe_load((ROOT / "services/procurement-service/docker-compose.yml").read_text())
    overlay = yaml.safe_load((ROOT / "services/procurement-service/docker-compose.gateway.yml").read_text())
    assert all(p.startswith("127.0.0.1:") for s in compose["services"].values() for p in s.get("ports", []))
    assert set(overlay["services"]) == {"procurement-service"}, "Do not attach PostgreSQL to the gateway network"
    assert "procurement-service" in overlay["services"]["procurement-service"]["networks"]["gateway"]["aliases"]
    assert overlay["networks"]["gateway"]["external"] is True

    result = subprocess.run(["go", "test", "-p", "1", "./internal/transport/httpapi", "-run",
                             "TestAdminReadContractExamples|TestInvalidCorrelationIDIsReplaced", "-count=1", "-v"],
                            cwd=ROOT / "services/procurement-service", capture_output=True, text=True)
    if result.returncode:
        raise RuntimeError(result.stdout + result.stderr)
    fixtures = [json.loads(line.split("PROCUREMENT_HTTP=", 1)[1]) for line in result.stdout.splitlines()
                if line.startswith("PROCUREMENT_HTTP=")]
    assert len(fixtures) >= 50, "Missing runtime response evidence"
    coverage = {path: set() for path in operations}
    for fixture in fixtures:
        path = urlsplit(fixture["path"]).path
        template = next((p for p in operations if re.fullmatch(re.sub(r"\{[^}]+\}", "[^/]+", p), path)), None)
        status = str(fixture["status"])
        if template and fixture["method"] == "GET":
            assert matches(path, "GET")
            response = resolve(spec, operations[template]["responses"][status])
            schema = response["content"]["application/json"]["schema"]
            coverage[template].add(status)
        else:
            assert fixture["status"] in (404, 405)
            schema = {"$ref": "#/components/schemas/Error"}
        validate(spec, schema, fixture["body"])
        expected_codes = {400: "INVALID_ARGUMENT", 401: "UNAUTHENTICATED", 403: "FORBIDDEN",
                          404: "NOT_FOUND", 405: "METHOD_NOT_ALLOWED", 500: "INTERNAL"}
        if fixture["status"] in expected_codes:
            assert fixture["body"]["code"] == expected_codes[fixture["status"]]
    for path, statuses in coverage.items():
        assert {"200", "401", "403"} <= statuses, (path, statuses)
    # Guard against weakening the Error contract or accepting null empty lists.
    invalid = {"code": "INTERNAL", "message": "Failed"}
    try:
        validate(spec, {"$ref": "#/components/schemas/Error"}, invalid)
    except ValidationError:
        pass
    else:
        raise AssertionError("Error.requestId is no longer required")
    for name in ("ProcurementSupplierPage", "ProcurementMaterialPage", "ProcurementOffers"):
        invalid = {"items": None}
        if name.endswith("Page"):
            invalid.update(limit=50, offset=0)
        try:
            validate(spec, {"$ref": "#/components/schemas/" + name}, invalid)
        except ValidationError:
            pass
        else:
            raise AssertionError("Null collection accepted: " + name)
    print(f"PASS: {len(operations)} read operations, {len(fixtures)} real HTTP/JWT payloads, examples, Kong allowlist and private network guards")
    if args.gateway_url:
        target = urlsplit(args.gateway_url)
        assert target.scheme == "http" and target.hostname == "127.0.0.1" and target.port
        assert not target.username and not target.password and target.path in ("", "/") and not target.query and not target.fragment
        base = args.gateway_url.rstrip("/")

        deadline = time.monotonic() + 30
        while True:
            try:
                with urlopen(base + "/", timeout=2) as response:
                    response.read()
                break
            except HTTPError:
                break  # Any HTTP response proves the proxy is listening.
            except URLError:
                if time.monotonic() >= deadline:
                    raise RuntimeError("Disposable Kong proxy did not start")
                time.sleep(0.2)

        def selected(path, method="GET"):
            # No Authorization, secrets or business payloads. The disposable
            # gateway must have allow_debug_header=on and no live upstreams.
            request = Request(base + path, method=method, headers={"Kong-Debug": "1"})
            try:
                response = urlopen(request, timeout=15)
            except HTTPError as error:
                response = error
            with response:
                response.read()
                return response.headers.get("Kong-Route-Name")

        for path in operations:
            concrete = re.sub(r"\{[^}]+\}", "10000000-0000-0000-0000-000000000001", path)
            assert selected(concrete) == route["name"], ("Kong failed to select read route", concrete)
            for method in ("POST", "PUT", "PATCH", "DELETE", "HEAD"):
                assert selected(concrete, method) != route["name"], ("Kong exposed method", concrete, method)
        for path in (PREFIX + "goods-receipts", PREFIX + "suppliers/x/contracts", "/health/ready",
                     PREFIX + "suppliers/", PREFIX + "suppliers-extra", PREFIX + "materials/x/offers/extra",
                     "/other" + PREFIX + "suppliers", PREFIX + "materials/x/preferred-supplier"):
            assert selected(path) != route["name"], ("Kong selected unexpected path", path)
        print("PASS: real Kong route selection: six GETs, thirty denied method combinations and eight denied paths (not service/deployment E2E)")


if __name__ == "__main__":
    main()
