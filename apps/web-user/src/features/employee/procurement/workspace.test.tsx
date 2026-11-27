import { afterEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { ProcurementWorkspace } from "./workspace";
import { ResourceOperations, RecordDetails, CreateOperations } from "./operations";
import { BusinessForm } from "./business-form";
import { formSpec } from "./forms";

const identity = vi.hoisted(() => ({ grants: [] as string[], actor: "11111111-1111-4111-8111-111111111111" }));
vi.mock("@/components/auth/employee-session-provider", () => ({ useEmployeeSession: () => ({ hasPermission: (permission: string) => identity.grants.includes(permission), session: { employee: { employeeId: identity.actor } } }) }));
afterEach(() => { identity.grants = []; });
describe("Procurement initial UI states", () => {
  it("shows pending real-data loading rather than demo rows", () => {
    identity.grants = ["PROCUREMENT_VIEW"];
    const html = renderToStaticMarkup(<ProcurementWorkspace />);
    expect(html).toContain("Đang tải dữ liệu"); expect(html).toContain("Phiếu nhận hàng");
    expect(html).not.toContain("SUP-DEMO"); expect(html).not.toContain("4030");
  });
  it("lets receiving-only staff look up a receipt without granting master-data access", () => {
    identity.grants = ["PROCUREMENT_RECEIVE"];
    const html = renderToStaticMarkup(<ProcurementWorkspace />);
    expect(html).toContain("Tra cứu phiếu nhận hàng"); expect(html).toContain("Chưa có quyền tra cứu danh mục/PO");
    expect(html).not.toContain("Danh mục <select");
  });
  it("does not turn manage or approve into read permission, while exposing permitted creates", () => {
    identity.grants = ["PROCUREMENT_MANAGE", "PROCUREMENT_APPROVE"];
    const html = renderToStaticMarkup(<ProcurementWorkspace />);
    expect(html).toContain("Chưa có quyền tra cứu danh mục/PO"); expect(html).not.toContain("Tra cứu phiếu nhận hàng");
    expect(html).toContain("Tạo nhà cung cấp"); expect(html).toContain("Tạo nguyên liệu");
    expect(html).toContain("Tạo đơn mua hàng"); expect(html).toContain("Hợp đồng");
  });
});

const other = "22222222-2222-4222-8222-222222222222";
const source = { id: other, version: 8, updated_at: "2026-11-28T11:22:33.123456Z", code: "SUP-01", supplier_id: other, status: "DRAFT" };
describe("Procurement operation controls and forms", () => {
  it("renders a business form rather than a JSON request editor, with immutable concurrency fields", () => {
    const html = renderToStaticMarkup(<BusinessForm spec={formSpec("supplier-edit", source)} onSuccess={async () => undefined} onCancel={() => undefined} />);
    expect(html).toContain("Sửa nhà cung cấp"); expect(html).toContain("Tên nhà cung cấp");
    expect(html).toContain('readOnly=""'); expect(html).toContain("2026-11-28T11:22:33.123456Z");
    expect(html).not.toContain("document_file_id"); expect(html).not.toContain("application/json");
  });
  it("shows approval to an independent approver, not to the creator", () => {
    identity.grants = ["PROCUREMENT_APPROVE"];
    const html = renderToStaticMarkup(<ResourceOperations kind="purchase-orders" row={{ ...source, status: "PENDING_APPROVAL", created_by: other }} />);
    expect(html).toContain("Duyệt PO"); expect(html).toContain("Từ chối PO"); expect(html).not.toContain("Sửa bản nháp PO");
    const self = renderToStaticMarkup(<ResourceOperations kind="purchase-orders" row={{ ...source, status: "PENDING_APPROVAL", created_by: identity.actor }} />);
    expect(self).toContain("không được tự duyệt"); expect(self).not.toContain(">Duyệt PO<");
  });
  it("shows contract and receiving controls only with the corresponding permissions", () => {
    identity.grants = ["PROCUREMENT_VIEW"];
    expect(renderToStaticMarkup(<ResourceOperations kind="contracts" row={source} />)).not.toContain("Kích hoạt hợp đồng");
    identity.grants = ["PROCUREMENT_MANAGE"];
    expect(renderToStaticMarkup(<ResourceOperations kind="contracts" row={source} />)).toContain("Kích hoạt hợp đồng");
    identity.grants = ["PROCUREMENT_RECEIVE"];
    const html = renderToStaticMarkup(<ResourceOperations kind="goods-receipts" row={source} />);
    expect(html).toContain("Kiểm nhận nguyên liệu"); expect(html).toContain("QUARANTINE");
    expect(html).not.toContain("Xử lý chênh lệch</button>");
  });
  it("exposes only unresolved variance controls and uses the receipt version", () => {
    identity.grants = ["PROCUREMENT_VARIANCE_RESOLVE"];
    const html = renderToStaticMarkup(<ResourceOperations kind="goods-receipts" row={{ ...source, variances: [{ id: identity.actor, status: "OPEN" }, { id: other, status: "RESOLVED" }] }} />);
    expect(html.match(/>Xử lý chênh lệch<\/button>/gu)).toHaveLength(1);
    expect(html).not.toContain("Kiểm nhận nguyên liệu");
  });
  it("does not expose creation based on role names and omits arbitrary contract document URLs", () => {
    identity.grants = ["ADMIN"];
    expect(renderToStaticMarkup(<CreateOperations onChanged={() => undefined} />)).toContain("Chưa có quyền tạo");
    const html = renderToStaticMarkup(<RecordDetails row={{ id: other, document_file_id: "https://evil.test", payment_terms: "<script>unsafe</script>" }} />);
    expect(html).not.toContain("evil.test"); expect(html).not.toContain("<script>"); expect(html).toContain("&lt;script&gt;");
  });
  it("requires explicit quantities for inspection and does not supply unverified dates", () => {
    const row = { ...source, lines: [{ id: other, quantity_received: "0.3", expected_open_quantity: "0.3" }] };
    const html = renderToStaticMarkup(<BusinessForm spec={formSpec("receipt-inspect", row)} onSuccess={async () => undefined} onCancel={() => undefined} />);
    expect(html).toContain("Số lượng chấp nhận"); expect(html).toContain("Số lượng từ chối");
    expect(html).toContain("Inventory"); expect(html).not.toContain("Ngày sản xuất");
  });
});
