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
    parser.add_argument("--web-forms", action="store_true", help="Also validate built Web form requests against all shared mutation schemas (requires installed Web dependencies and Node)")
    args = parser.parse_args()
    spec = yaml.safe_load((ROOT / "docs/03_api_specs/admin-stable.openapi.yaml").read_text(encoding="utf-8"))
    kong = yaml.safe_load((ROOT / "services/api-gateway/config/kong.yml").read_text(encoding="utf-8"))
    services = [s for s in kong["services"] if s["name"] == "procurement-service"]
    assert len(services) == 1
    service = services[0]
    assert service["url"] == "http://procurement-service:8008"
    assert service["retries"] == 0, "Do not silently add transport replay"
    assert len(service["routes"]) == 12
    assert "plugins" not in service, "Do not inject user/internal credentials"
    patterns = []
    for route in service["routes"]:
        assert route["strip_path"] is False and "plugins" not in route
        assert set(route["methods"]) in ({"GET", "OPTIONS"}, {"POST", "OPTIONS"}, {"PUT", "OPTIONS"}, {"PATCH", "OPTIONS"})
        for path in route["paths"]:
            assert path.startswith("~/") and path.endswith("$")
            assert "(?" not in path, "Kong expression regex does not support look-around"
            patterns.append((re.compile(path[1:]), route))

    def matches(path, method):
        return [route["name"] for pattern, route in patterns if method in route["methods"] and pattern.fullmatch(path)]

    operations = {(path, method.upper()): operation for path, item in spec["paths"].items() if path.startswith(PREFIX)
                  for method, operation in item.items() if method in ("get", "post", "put", "patch")}
    assert len(operations) == 37, "Keep the contracted gateway surface explicit"
    if args.web_forms:
        result = subprocess.run(["node", "node_modules/vitest/vitest.mjs", "run",
                                 "src/features/employee/procurement/forms.test.ts", "--silent=false", "--reporter=dot"],
                                cwd=ROOT / "apps/web-user", capture_output=True, text=True, encoding="utf-8")
        if result.returncode:
            raise RuntimeError(result.stdout + result.stderr)
        forms = [json.loads(line.split("PROCUREMENT_FORM=", 1)[1]) for line in result.stdout.splitlines()
                 if line.strip().startswith("PROCUREMENT_FORM=")]
        covered = set()
        for form in forms:
            form_matches = [key for key in operations if form["method"] == key[1]
                            and re.fullmatch(re.sub(r"\{[^}]+\}", "[^/]+", key[0]), form["path"])]
            assert len(form_matches) == 1, form["action"]
            key = form_matches[0]
            validate(spec, operations[key]["requestBody"]["content"]["application/json"]["schema"], form["request"])
            covered.add(key)
        assert covered == {key for key in operations if key[1] != "GET"}
        print("PASS: all 25 Web business-form mutation payloads match the shared JSON Schemas")
    for facade, prefix, security in (
        ("web", "/api/admin/procurement/", [{"procurementBrowserSession": []}, {"procurementBrowserOidc": []}]),
        ("identity", "/api/v1/admin/procurement-api/", [{"procurementBrowserSession": []}, {"bearerAuth": []}]),
    ):
        facade_operations = {(path, method.upper()): op for path, item in spec["paths"].items() if path.startswith(prefix)
                             for method, op in item.items() if method in ("get", "post", "put", "patch")}
        assert len(facade_operations) == 37, "No missing operations or planning routes in facade"
        for (path, method), operation in operations.items():
            alias = facade_operations[(prefix + path.removeprefix(PREFIX), method)]
            assert alias["security"] == security and alias["servers"][0]["url"] == "/"
            for grants in ("x-required-grants", "x-required-grants-any"):
                assert alias.get(grants) == operation.get(grants), (facade, path, grants)
            for status, response in operation["responses"].items():
                if str(status) == "404":
                    continue
                assert resolve(spec, alias["responses"][status]) == resolve(spec, response)
            assert {"502", "503", "404"}.issubset(alias["responses"])
            if method != "GET":
                assert resolve(spec, alias["requestBody"]) == operation["requestBody"]
                assert {"413", "415"}.issubset(alias["responses"])
                key = next(resolve(spec, p) for p in alias["parameters"] if resolve(spec, p)["name"] == "Idempotency-Key")
                assert key["required"] is True
    for (path, method), operation in operations.items():
        assert operation["security"] == [{"bearerAuth": []}], "No implicit cookie authentication"
        if "/goods-receipts" in path:
            scope = "warehouse_code" if path == PREFIX+"goods-receipts" else "receiptWarehouse"
            expected = {"PROCUREMENT_RECEIVE#GLOBAL", "PROCUREMENT_RECEIVE#WAREHOUSE#<"+scope+">"}
            if method == "GET":
                expected.add("PROCUREMENT_VIEW#GLOBAL")
            if "/variances/" in path:
                expected = {"PROCUREMENT_VARIANCE_RESOLVE#GLOBAL", "PROCUREMENT_VARIANCE_RESOLVE#WAREHOUSE#<receiptWarehouse>"}
            assert set(operation["x-required-grants-any"]) == expected
        else:
            permission = "PROCUREMENT_MANAGE#GLOBAL" if "/contracts" in path else "PROCUREMENT_VIEW#GLOBAL"
            if method in ("POST","PUT","PATCH"):
                permission = "PROCUREMENT_APPROVE#GLOBAL" if operation.get("x-po-action") in ("APPROVE","REJECT") else "PROCUREMENT_MANAGE#GLOBAL"
            assert operation["x-required-grants"] == [permission]
        concrete = re.sub(r"\{[^}]+\}", "10000000-0000-0000-0000-000000000001", path)
        assert len(matches(concrete, method)) == 1 and matches(concrete, "OPTIONS")
        allowed_methods = {m for p, m in operations if p == path}
        for denied in {"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"} - allowed_methods:
            assert not matches(concrete, denied), (concrete, denied)
        if method in ("POST","PUT","PATCH"):
            key = next(resolve(spec, parameter) for parameter in operation["parameters"]
                       if resolve(spec, parameter)["name"] == "Idempotency-Key")
            assert key["required"] is True and key["schema"]["minLength"] == 16
            request = operation["requestBody"]["content"]["application/json"]
            validate(spec, request["schema"], request["example"])
        for response in operation["responses"].values():
            media = resolve(spec, response)["content"]["application/json"]
            examples = [e["value"] for e in media.get("examples", {}).values()]
            if "example" in media:
                examples.append(media["example"])
            assert examples, "Each documented state needs an example"
            for example in examples:
                validate(spec, media["schema"], example)

    for path in ("/health/live", "/health/ready", PREFIX + "purchase-orders/extra/unsupported",
                 PREFIX + "goods-receipts", PREFIX + "supply-plans", PREFIX + "contracts/x/extra",
                 PREFIX + "suppliers/x/contracts/extra", PREFIX + "suppliers/x/status",
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
                             "TestAdminReadContractExamples|TestInvalidCorrelationIDIsReplaced|TestReceivingContractExamples|TestPOActionsContractExamples|TestPORevisionContractExamples|TestPODraftAndListContractExamples|TestVarianceResolutionContractExamples|TestMasterDataCreateContractExamples|TestMasterDataMutationContractExamples|TestSupplierContractContractExamples", "-count=1", "-v"],
                            cwd=ROOT / "services/procurement-service", capture_output=True, text=True)
    if result.returncode:
        raise RuntimeError(result.stdout + result.stderr)
    fixtures = [json.loads(line.split("PROCUREMENT_HTTP=", 1)[1]) for line in result.stdout.splitlines()
                if line.startswith("PROCUREMENT_HTTP=")]
    assert len(fixtures) >= 490, "Missing runtime response evidence"
    coverage = {key: set() for key in operations}
    for fixture in fixtures:
        path = urlsplit(fixture["path"]).path
        template = next((key for key in operations if fixture["method"] == key[1] and re.fullmatch(re.sub(r"\{[^}]+\}", "[^/]+", key[0]), path)), None)
        status = str(fixture["status"])
        if template:
            assert matches(path, fixture["method"])
            response = resolve(spec, operations[template]["responses"][status])
            schema = response["content"]["application/json"]["schema"]
            coverage[template].add(status)
            if fixture["method"] in ("POST","PUT","PATCH") and fixture["status"] < 300:
                validate(spec, operations[template]["requestBody"]["content"]["application/json"]["schema"], fixture["request"])
        else:
            assert fixture["status"] in (404, 405)
            schema = {"$ref": "#/components/schemas/Error"}
        validate(spec, schema, fixture["body"])
        expected_codes = {400: {"INVALID_ARGUMENT", "INVALID_JSON"}, 401: {"UNAUTHENTICATED"}, 403: {"FORBIDDEN"},
                          404: {"NOT_FOUND"}, 405: {"METHOD_NOT_ALLOWED"}, 409: {"CONFLICT", "INVALID_STATE"}, 500: {"INTERNAL"}}
        if fixture["status"] in expected_codes:
            assert fixture["body"]["code"] in expected_codes[fixture["status"]]
    for key, statuses in coverage.items():
        assert {"401", "403"} <= statuses and {"200", "201"} & statuses, (key, statuses)
        if key[1] in ("POST","PUT","PATCH"):
            assert {"400", "409"} <= statuses, (key, statuses)
    # Guard against weakening the Error contract or accepting null empty lists.
    invalid = {"code": "INTERNAL", "message": "Failed"}
    try:
        validate(spec, {"$ref": "#/components/schemas/Error"}, invalid)
    except ValidationError:
        pass
    else:
        raise AssertionError("Error.requestId is no longer required")
    for name in ("ProcurementSupplierPage", "ProcurementMaterialPage", "ProcurementOffers", "ProcurementPORevisionPage", "ProcurementPOPage", "ProcurementContractPage"):
        invalid = {"items": None}
        if name.endswith("Page"):
            invalid.update(limit=50, offset=0)
        try:
            validate(spec, {"$ref": "#/components/schemas/" + name}, invalid)
        except ValidationError:
            pass
        else:
            raise AssertionError("Null collection accepted: " + name)
    print(f"PASS: {len(operations)} operations, {len(fixtures)} real HTTP/JWT payloads, examples, Kong allowlist and private network guards")
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

        for path, method in operations:
            concrete = re.sub(r"\{[^}]+\}", "10000000-0000-0000-0000-000000000001", path)
            assert selected(concrete, method) in matches(concrete, method), ("Kong failed to select contracted route", concrete)
            allowed_methods = {m for p, m in operations if p == path}
            for denied in {"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"} - allowed_methods:
                assert selected(concrete, denied) not in {r["name"] for r in service["routes"]}, ("Kong exposed method", concrete, denied)
        for path in (PREFIX + "goods-receipts", PREFIX + "suppliers/x/contracts/extra", "/health/ready",
                     PREFIX + "suppliers/", PREFIX + "suppliers-extra", PREFIX + "materials/x/offers/extra",
                     "/other" + PREFIX + "suppliers", PREFIX + "materials/x/preferred-supplier"):
            assert selected(path) not in {r["name"] for r in service["routes"]}, ("Kong selected unexpected path", path)
        for path in (PREFIX+"purchase-orders/x/revisions/extra", PREFIX+"purchase-orders/x/supplier-response/extra", PREFIX+"goods-receipts/x/variances/y/resolve/extra"):
            assert selected(path, "POST") not in {r["name"] for r in service["routes"]}, path
        print("PASS: real Kong selection: 37 operations, 167 denied methods and 11 denied paths (not service/deployment E2E)")


if __name__ == "__main__":
    main()
