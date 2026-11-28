"""Export six standalone Markdown owner briefs from a pinned documentation commit.

Requires PyYAML. Source contracts stay authoritative; exports are distribution
snapshots and must be regenerated, not edited independently.
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import re
import subprocess
from pathlib import Path
from urllib.parse import quote

import yaml


ROOT = Path(__file__).resolve().parents[1]
BASE = "docs/03_api_specs/"
OUTPUT = ROOT / BASE / "traceability-owner-handoffs"
REPOSITORY = "https://github.com/hvduong2k5/DUT.PBL6"
HANDOFF = BASE + "traceability-workflow-peer-handoff.md"
ACCEPTANCE = BASE + "traceability-workflow-acceptance.md"
INTEGRATION = BASE + "traceability-workflow-draft.openapi.yaml"
ADMIN = BASE + "traceability-scoped-admin-draft.openapi.yaml"
BFF = BASE + "traceability-workflow-bff-draft.openapi.yaml"
WEB = BASE + "traceability-web-bff-draft.openapi.yaml"
LOCAL_REPORT = "docs/04_testing/traceability-service/12_full_workflow_2026-11-28.md"
CUSTOMER_STABLE = BASE + "customer-stable.openapi.yaml"
STABLE = BASE + "admin-stable.openapi.yaml"
LEGACY = BASE + "traceability-service.openapi.yaml"
BATCH_PROTO = BASE + "traceability-workflow-candidates/batch_source.proto"
PACKING_PROTO = BASE + "traceability-workflow-candidates/packing.proto"
SUPPLEMENT = BASE + "traceability-workflow-supplement.md"
GRPC_GUIDE = BASE + "traceability-workflow-grpc.md"
MEDIA_PROTO = "packages/proto/media/v1/media_service.proto"
CATALOG_PROTO = "packages/proto/catalog/v1/catalog.proto"
IDENTITY_PROTO = "packages/proto/identity/v1/staff_authorization.proto"
LINKS_PROTO = "packages/proto/traceability/v1/batch_links.proto"


class PlainDumper(yaml.SafeDumper):
    def ignore_aliases(self, data):
        return True


class Sources:
    def __init__(self, commit: str):
        self.commit = commit
        self.texts: dict[str, str] = {}
        self.documents: dict[str, dict] = {}

    def text(self, path: str) -> str:
        if path not in self.texts:
            result = subprocess.run(
                ["git", "show", f"{self.commit}:{path}"], cwd=ROOT,
                check=True, capture_output=True,
            )
            self.texts[path] = result.stdout.decode("utf-8").replace("\r\n", "\n")
        return self.texts[path]

    def document(self, path: str) -> dict:
        if path not in self.documents:
            self.documents[path] = yaml.safe_load(self.text(path))
        return self.documents[path]

    def url(self, path: str) -> str:
        file, separator, fragment = path.partition("#")
        return f"{REPOSITORY}/blob/{self.commit}/{quote(file, safe='/')}" + (
            "#" + fragment if separator else ""
        )

    def target(self, file: str, reference: str):
        other, _, pointer = reference.partition("#")
        if other:
            resolved = (ROOT / file).parent.joinpath(other).resolve()
            file = resolved.relative_to(ROOT).as_posix()
        node = self.document(file)
        parts = [p.replace("~1", "/").replace("~0", "~") for p in pointer.lstrip("/").split("/")] if pointer else []
        for part in parts:
            node = node[part]
        return file, node, parts

    def pin_links(self, text: str, file: str) -> str:
        def replace(match):
            target = match.group(2)
            if "://" in target or target.startswith("#"):
                return match.group(0)
            name, separator, fragment = target.partition("#")
            path = ((ROOT / file).parent / name).resolve().relative_to(ROOT).as_posix()
            return f"[{match.group(1)}]({self.url(path + ('#' + fragment if separator else ''))})"
        return re.sub(r"\[([^\]]+)\]\(([^)]+)\)", replace, text)


class Bundle:
    """Keep shared components while rewriting every reference to a local one."""

    def __init__(self, sources: Sources, owner: str):
        self.sources = sources
        self.components: dict[str, dict] = {}
        self.paths: dict[str, dict] = {}
        self.files: set[str] = set()
        self.owner = owner

    @staticmethod
    def prefix(file: str) -> str:
        labels = {CUSTOMER_STABLE: "MediaStable", ADMIN: "Scoped", INTEGRATION: "Workflow", BFF: "IdentityBff", STABLE: "AdminStable", LEGACY: "Legacy"}
        return labels.get(file, re.sub(r"\W+", "_", Path(file).stem))

    def component(self, file: str, section: str, name: str) -> str:
        self.files.add(file)
        key = self.prefix(file) + "_" + name
        destination = self.components.setdefault(section, {})
        if key not in destination:
            destination[key] = {}
            destination[key] = self.convert(self.sources.document(file)["components"][section][name], file)
        return key

    def convert(self, value, file: str):
        if isinstance(value, list):
            return [self.convert(item, file) for item in value]
        if not isinstance(value, dict):
            return value
        value = copy.deepcopy(value)
        if "$ref" in value:
            target_file, target, parts = self.sources.target(file, value["$ref"])
            if len(parts) >= 3 and parts[0] == "components":
                key = self.component(target_file, parts[1], parts[2])
                suffix = "".join("/" + p.replace("~", "~0").replace("/", "~1") for p in parts[3:])
                result = {"$ref": f"#/components/{parts[1]}/{key}{suffix}"}
            else:
                result = self.convert(target, target_file)
            result.update({k: self.convert(v, file) for k, v in value.items() if k != "$ref"})
            return result
        if value.pop("nullable", False):
            value["type"] = [value["type"], "null"]
        return {key: self.convert(item, file) for key, item in value.items()}

    def add(self, file: str, path: str, method: str):
        self.files.add(file)
        item = self.sources.document(file)["paths"][path]
        while "$ref" in item:
            file, item, _ = self.sources.target(file, item["$ref"])
            self.files.add(file)
        operation = copy.deepcopy(item[method])
        security = operation.pop("security", self.sources.document(file).get("security", []))
        operation = self.convert(operation, file)
        operation.setdefault("summary", operation.get("operationId", f"{method.upper()} {path}"))
        operation["security"] = [
            {self.component(file, "securitySchemes", name): scopes for name, scopes in requirement.items()}
            for requirement in security
        ]
        operation.setdefault("servers", item.get("servers", self.sources.document(file).get("servers", [])))
        if file == LEGACY and path == "/api/v2/trace/{qr_code}":
            # The v2 public path belongs to Kong, not the service's v1 listener.
            self.sources.text("services/api-gateway/docker-compose.yml")
            operation["servers"] = [{"url": "http://127.0.0.1:8000"}]
            operation["description"] = operation.get("description", "") + (
                " The local server example is Kong's configured proxy port. "
                "Configure the gateway origin per environment; do not call Traceability port 8003 with this v2 path."
            )
        operation["tags"] = [self.owner]
        destination = self.paths.setdefault(path, {})
        if "parameters" in item:
            destination["parameters"] = self.convert(item["parameters"], file)
        if method in destination:
            raise ValueError(f"Duplicate operation {method} {path}")
        destination[method] = operation

    def receipt_delta(self):
        operation = self.paths["/api/v1/admin/inventory/workbench/receipts"]["post"]
        media = operation["responses"]["200"]["content"]["application/json"]
        schema = media["schema"]
        if "$ref" in schema:
            _, section, name = schema["$ref"].lstrip("#/").split("/")
            schema = copy.deepcopy(self.components[section][name])
        additions = self.sources.document(INTEGRATION)["components"]["schemas"]["ReceiptBindingExtension"]["properties"]
        schema["properties"].update(copy.deepcopy(additions))
        media["schema"] = schema
        operation["description"] = operation.get("description", "") + (
            " Fresh APPLIED receipts include batchId, batchCode, skuCode and warehouseId from the committed transaction. "
            "Legacy idempotent replay may omit them; preserve the receipt result and do not repeat the stock mutation."
        )

    def dump(self) -> str:
        result = {
            "openapi": "3.1.0",
            "info": {
                "title": f"PBL6 {self.owner} traceability handoff",
                "version": "0.4.0-draft",
                "license": {"name": "Proprietary", "identifier": "LicenseRef-Proprietary"},
                "description": f"Distribution snapshot from documentation commit {self.sources.commit}. Direct implementation target, not runtime acceptance. All references are local to this embedded contract.",
            },
            "tags": [{"name": self.owner, "description": "Owner-specific provider and dependency contracts"}],
            "servers": [{"url": "http://127.0.0.1"}],
            "paths": self.paths,
            "components": self.components,
            "x-source-commit": self.sources.commit,
        }
        return yaml.dump(result, Dumper=PlainDumper, allow_unicode=True, sort_keys=False, width=110)


def section(text: str, number: int) -> str:
    match = re.search(rf"^## {number}\. .*?(?=^## \d+\. |\Z)", text, re.M | re.S)
    if not match:
        raise ValueError(f"Missing handoff section {number}")
    return match.group(0).strip()


def test_rows(text: str, prefixes: list[str]) -> str:
    rows = []
    for line in text.splitlines():
        match = re.match(r"\| ([A-Z]+(?:-[A-Z]+)*-\d+) \|", line)
        if match and any(match.group(1).startswith(p) for p in prefixes):
            rows.append(line)
    if not rows:
        raise ValueError(f"No acceptance rows for {prefixes}")
    return "| Test ID | Ca kiểm | Expected |\n| --- | --- | --- |\n" + "\n".join(rows)


OWNERS = [
    ("01-inventory.md", "Inventory", "INV-TRC-01", 1, ["INV-", "T-SCOPE-04"], [BATCH_PROTO]),
    ("02-catalog.md", "Catalog / Media", "CAT-TRC-01", 6, ["CAT-", "MED-", "T-CTX-", "GRPC-CAT-"], [CATALOG_PROTO, MEDIA_PROTO, "packages/proto/common/v1/types.proto", "packages/proto/common/v1/errors.proto"]),
    ("03-identity.md", "Identity / Keycloak", "IAM-TRC-01", 2, ["IAM-", "T-SCOPE-01", "T-SCOPE-02", "T-SCOPE-03", "WEB-05", "GRPC-IAM-"], [IDENTITY_PROTO]),
    ("04-fulfillment.md", "Fulfillment", "FUL-TRC-01", 3, ["FUL-", "EVT-", "ORD-07"], [PACKING_PROTO]),
    ("05-order.md", "Order", "ORD-TRC-01", 4, ["ORD-", "EVT-", "GRPC-RES-"], [PACKING_PROTO, LINKS_PROTO]),
    ("06-web-gateway.md", "Frontend / Gateway", "WEB-TRC-01", 5, ["WEB-", "T-QR-", "ORD-04", "T-CTX-"], []),
]


def render(sources: Sources, owner) -> str:
    _, title, task, number, tests, protos = owner
    handoff = sources.text(HANDOFF)
    specific = section(handoff, number)
    if number == 6:
        specific = specific.split("### Các miền khác", 1)[0].strip()
    specific = re.sub(r"^## \d+\. [^\n]+\n+", "", specific, count=1)
    specific = sources.pin_links(specific, HANDOFF)
    common = sources.pin_links(section(handoff, 0), HANDOFF)
    common = re.sub(r"^## \d+\. [^\n]+\n+", "", common, count=1)
    bundle = Bundle(sources, title)
    if number == 1:
        bundle.add(STABLE, "/api/v1/admin/inventory/workbench", "get")
        bundle.add(STABLE, "/api/v1/admin/inventory/workbench/receipts", "post")
        bundle.receipt_delta()
    elif number == 2:
        for path, item in sources.document(BFF)["paths"].items():
            for method in item:
                if method in {"get", "post", "patch"}:
                    bundle.add(BFF, path, method)
    elif number == 4:
        bundle.add(INTEGRATION, "/api/v1/orders/{order_id}/traceability", "get")
    elif number == 5:
        for path in sources.document(ADMIN)["paths"]:
            file, item = ADMIN, sources.document(ADMIN)["paths"][path]
            while "$ref" in item:
                file, item, _ = sources.target(file, item["$ref"])
            for method in item:
                if method in {"get", "post", "patch"}:
                    bundle.add(ADMIN, path, method)
        bundle.add(INTEGRATION, "/api/v1/orders/{order_id}/traceability", "get")
        bundle.add(LEGACY, "/api/v2/trace/{qr_code}", "get")
        for path, item in sources.document(WEB)["paths"].items():
            for method in item:
                if method in {"get", "post", "patch"}:
                    bundle.add(WEB, path, method)

    if number in {5, 6}:
        for path in ["/api/v1/media/upload-sessions", "/api/v1/media/upload-sessions/{uploadSessionId}/complete"]:
            bundle.add(CUSTOMER_STABLE, path, "post")

    source_list = [HANDOFF, ACCEPTANCE, SUPPLEMENT,
                   BASE + "traceability-workflow-service-spec.md", BASE + "traceability-workflow-review-2026-11-25.md", GRPC_GUIDE, LOCAL_REPORT]
    source_list += sorted(bundle.files) + protos
    if number == 5:
        source_list += [BFF, WEB, "services/api-gateway/config/kong.yml"]
    source_list = list(dict.fromkeys(source_list))
    rows = "\n".join(f"| `{path}` | [Mở đúng commit]({sources.url(path)}) |" for path in source_list)
    checksum = hashlib.sha256("".join(path + "\n" + sources.text(path) for path in source_list).encode("utf-8")).hexdigest()
    result = f"""# PBL6 — Bàn giao {title}: {task}

Version nghiệp vụ/contract: **0.4.0-draft** · Snapshot tài liệu: **`{sources.commit}`**.

**Bản 0.4.0 thay phiếu 0.3.0/0.2.2:** theo Catalog/Media đã merge trong PR #175 và plan Dương; không triển khai proposal GetSkuBinding đã gỡ. Web/BFF vẫn REST. Identity current-grants và Order resolver vẫn là phần việc riêng, chưa có trong plan Dương.

**Checkpoint28/11:** phần bổ sung đã có trên nhánh này; đọc report12 để tránh code trùng, xem kết quả toàn luồng và giới hạn nghiệm thu. Owner bảo đảm tương thích khi ghép, không thay phần pricing/checkout trong plan Order Dương.

File này là bản `.md` độc lập để gửi qua Zalo. Đề bài, yêu cầu đầu ra, test và contract liên quan được nhúng bên dưới; không cần đọc lịch sử chat hoặc chờ phản hồi ACCEPT để bắt đầu code. Draft không đồng nghĩa chức năng đã chạy hoặc đã deploy.

## Nguồn tài liệu và cách lấy về

- Repository: [{REPOSITORY}]({REPOSITORY}). Nhánh tài liệu: `codex/traceability-order-flow`.
- **Đọc theo commit đã ghim**, không tự thay bằng HEAD của dev hoặc nhánh tài liệu đang tiếp tục thay đổi. Đây là nguồn tài liệu; owner giữ nhánh làm việc riêng cho service, không phải checkout đè công việc của mình.
- Có thể đọc toàn bộ đặc tả qua các link dưới. Người truy cập repository riêng tư cần quyền hiện có của nhóm. Contract nhúng trong file này vẫn đọc được khi chưa mở được link.

| Tài liệu liên quan trong repository | Link bất biến |
| --- | --- |
{rows}

Trong repository đã clone, các lệnh đọc sau không checkout/reset hay ghi đè source đang làm:

```powershell
git fetch origin codex/traceability-order-flow
git show {sources.commit}:docs/03_api_specs/traceability-workflow-grpc.md
git show {sources.commit}:docs/03_api_specs/traceability-workflow-peer-handoff.md
```

## Bắt đầu thực hiện

{common}

## Phần việc của {title}

{specific}

## Luồng chung cần giữ đúng

Nhập kho commit tại Inventory → nhân viên mở batch context và lập/xem hồ sơ Traceability → người độc lập duyệt, sau đó publish → Web tạo hình QR từ public URL để tải/in → dán đúng lô. Khách scan public không cần login. Xem từ đơn dùng lô của exact sealed packing proof, không dùng tồn hiện tại. Lỗi hồ sơ không rollback receipt; lỗi truy xuất không chặn lifecycle/payment/shipping của Order. Không publish/kích hoạt QR từ event packing.

Nhân viên chỉ thao tác trong kho được giao. Role là bundle permission; scope phải gắn đúng permission/warehouse. Không bổ sung GLOBAL để né thiếu scoped support. Hiện manifest Order một SKU/một line; một line có thể nhiều lô. Không suy ID từ tên, mã lô hiển thị hoặc index UI.

## Test bắt buộc cho task này

{test_rows(sources.text(ACCEPTANCE), tests)}

Nộp kết quả cuối/PR: OUT đã đáp ứng, SHA, phần tái sử dụng/bổ sung, mapping adapter nếu có, PASS/FAIL/SKIPPED từng test và bằng chứng đã lọc. Chưa có peer chạy thật vẫn code/test bằng fixture đúng contract, ghi `IMPLEMENTED_NOT_INTEGRATED`; không báo full E2E bằng mock. Chỉ báo blocker vượt scope, không gửi phương án rồi chờ duyệt trước sửa code.

Đọc AGENTS.md và rule service trong repository trước khi sửa. Commit mới dùng tiếng Anh, giữ một type/scope; contract thay trước hoặc cùng code, không sửa migration cũ/generated rows sai quy trình. Không tự push dev/release hoặc deploy production; tạo PR theo quy trình nhóm.
"""
    supplement = sources.pin_links(sources.text(SUPPLEMENT), SUPPLEMENT).split("\n", 1)[1].strip()
    result += "\n## Phần bổ sung riêng và ranh giới phối hợp\n\n" + supplement + "\n"
    guide = sources.pin_links(sources.text(GRPC_GUIDE), GRPC_GUIDE).split("\n", 1)[1].strip()
    result += "\n## Quy định transport mới và contract gRPC\n\n" + guide + "\n"
    if bundle.paths:
        result += "\n## Contract HTTP nhúng — không cần tải thêm file để đọc schema\n\n"
        result += "Khối YAML dưới đây là bản đóng gói độc lập; `$ref` đã đổi thành nội bộ khối này. Có thể chép nguyên khối vào `owner-contract.openapi.yaml` để lint. Server URLs là ví dụ cấu hình; giữ path/method/header/field/status theo contract. Không tạo endpoint mới nếu lớp hiện có đã trả đúng đầu ra; khác wire thì làm adapter trong phần mình phụ trách.\n\n```yaml\n"
        result += bundle.dump() + "```\n"
    if number == 5:
        result += "\n## Mapping cookie transport do Identity cung cấp\n\n| BFF route | Backend route qua Kong |\n| --- | --- |\n| `/api/v1/admin/trace-api/...` | `/api/v2/admin/trace/...` |\n| `/api/v1/admin/inventory-api/workbench` | `/api/v1/admin/inventory/workbench` |\n| `/api/v1/admin/inventory-api/workbench/receipts` | `/api/v1/admin/inventory/workbench/receipts` |\n\nCookie mode chuyển `OMAMA_SESSION` server-side qua Identity; forward Origin đã kiểm tra ở mutation, giữ Idempotency-Key. Bearer mode lấy credential tin cậy trong server, không cho client gán user/role headers. Internal resolver và Catalog binding không route public.\n"
    for proto in protos:
        instruction = ("Candidate full definition: compile riêng, cập nhật canonical/copies/codegen khi implement; không compile cạnh canonical cùng message." if proto.startswith(BASE) else "Shared canonical integration-target: thêm file này vào pipeline proto/copies/codegen của service, giữ mọi method/field cũ; không chép thêm một definition trùng.")
        result += f"\n## Proto target nhúng: `{proto}`\n\n{instruction} Không hand-edit generated code.\n\n```proto\n{sources.text(proto).rstrip()}\n```\n"
    result += f"\n## Tính toàn vẹn bản gửi\n\nNguồn commit: `{sources.commit}`. SHA-256 tập source: `{checksum}`. Đây là snapshot phân phối được sinh từ nguồn đã commit, không sửa riêng file gửi Zalo rồi coi là contract mới. Nếu scope/contract đổi, owner nghiệp vụ phát lại snapshot version/commit mới. Các link nguồn đặc tả và nội dung nhúng dùng cùng commit; liên kết bằng chứng Git/PR lịch sử ghi SHA provider riêng trong phần đối chiếu. Không có token/password trong file.\n"
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-commit", required=True)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    if not re.fullmatch(r"[0-9a-fA-F]{7,40}", args.source_commit):
        parser.error("--source-commit must be a Git commit hash")
    commit = subprocess.run(["git", "rev-parse", "--verify", args.source_commit + "^{commit}"], cwd=ROOT, check=True, capture_output=True, text=True).stdout.strip()
    sources = Sources(commit)
    if sources.document(INTEGRATION)["info"]["version"] != "0.4.0-draft":
        parser.error("This exporter targets contract 0.4.0-draft; update the exporter before exporting another version")
    outputs = {owner[0]: render(sources, owner) for owner in OWNERS}
    outputs["README.md"] = "# Bản Markdown gửi riêng cho từng owner\n\n" + (
        f"Source commit `{commit}`, target0.4.0-draft. Bản này thay phiếu0.3.0/0.2.2, theo canonical Catalog/Media của Dương trong PR #175; giữ REST Web và BFF. Gửi đúng một file bên dưới cho từng owner; mỗi file có đề bài, link theo commit, contract nhúng và test. Không cần gửi toàn bộ lịch sử chat. Đây là snapshot phân phối, không phải spec độc lập để sửa tay.\n\n"
    ) + "\n".join(f"- [{title} — {task}]({filename})" for filename, title, task, *_ in OWNERS) + (
        f"\n\nSinh lại từ repo root (Python có PyYAML): `python scripts/export-traceability-owner-handoffs.py --source-commit {commit}`. Kiểm snapshot khớp source bằng thêm `--check`. Không tạo PDF/Word/ZIP.\n"
    )
    if not args.check:
        OUTPUT.mkdir(parents=True, exist_ok=True)
    for name, text in outputs.items():
        path = OUTPUT / name
        if args.check:
            if not path.is_file() or path.read_text(encoding="utf-8") != text:
                raise SystemExit(f"Snapshot mismatch: {path.relative_to(ROOT)}")
        else:
            path.write_text(text, encoding="utf-8", newline="\n")
    print(f"{'Verified' if args.check else 'Exported'} six standalone owner Markdown briefs and index from {commit[:8]}.")


if __name__ == "__main__":
    main()
