import { describe, expect, it } from "vitest";
import { availableActions, buildPayload, formSpec, type Draft, type RecordData } from "./forms";
import { procurementRoute } from "./routes";

export const fixtureID = "11111111-1111-4111-8111-111111111111";
export const otherID = "22222222-2222-4222-8222-222222222222";
export const fixtureSource: RecordData = { id: fixtureID, supplier_id: otherID, receipt_id: otherID, version: 7, updated_at: "2026-11-28T10:11:12.123456Z", status: "ACTIVE", code: "SUP-UI", name: "Supplier", supplier_type: "COMPANY", base_unit: "KG", lines: [{ id: otherID, material_id: fixtureID, quantity: "0.300000", quantity_received: "0.300000", expected_open_quantity: "0.300000" }] };
export const writeActions = ["supplier-create", "supplier-edit", "supplier-status", "material-create", "material-edit", "material-status", "material-preference", "offer-create", "offer-status", "contract-create", "contract-activate", "contract-terminate", "po-create", "po-edit", "po-revise", "po-submit", "po-approve", "po-reject", "po-send", "po-response", "po-cancel", "po-close", "receipt-create", "receipt-inspect", "variance-resolve"];
const values: Record<string, string> = { code: "SUP-UI", name: "Supplier", supplier_type: "COMPANY", base_unit: "KG", unit: "KG", currency: "VND", supplier_id: otherID, material_id: fixtureID, purchase_order_id: fixtureID, purchase_order_line_id: otherID, goods_receipt_line_id: otherID, status: "ACTIVE", active: "true", unit_price_minor: "12000", minimum_quantity: "0.01", lead_time_days: "2", effective_from: "2026-11-28", effective_to: "2026-12-31", expected_delivery_date: "2026-12-01", contract_number: "CT-UI-1", scope: "Material supply", payment_terms: "Net 30", reason: "Reviewed by operator", note: "Supplier requests revised terms", response: "CHANGE_REQUESTED", warehouse_code: "WH-01", quantity: "0.300000", quantity_received: "0.300000", quantity_accepted: "0.100000", quantity_rejected: "0.200000", inspection_notes: "Rejected after inspection", decision: "ACCEPT_VARIANCE" };
export function fixtureDraft(action: string): Draft {
  const spec = formSpec(action, fixtureSource);
  return { values: Object.fromEntries(spec.fields.map((field) => [field.name, field.locked ? spec.initial[field.name] : values[field.name] ?? ""])), lines: spec.lineFields ? [Object.fromEntries(spec.lineFields.map((field) => [field.name, values[field.name] ?? ""]))] : [] };
}
describe("Procurement business form payloads", () => {
  it("covers every write route without exposing planning or attachment fields", () => {
    const operations = new Set<string>();
    for (const action of writeActions) {
      const spec = formSpec(action, fixtureSource), payload = buildPayload(spec, fixtureDraft(action));
      expect(procurementRoute(spec.method, spec.path), action).toBe(true);
      operations.add(`${spec.method} ${spec.path}`);
      expect(payload).not.toHaveProperty("actor"); expect(payload).not.toHaveProperty("document_file_id");
      expect(JSON.stringify(payload)).not.toContain("forecast");
      // The repository contract verifier validates these exact built requests
      // against the shared JSON Schemas with --web-forms.
      console.log("PROCUREMENT_FORM=" + JSON.stringify({ action, method: spec.method, path: "/api/v1/admin/procurement/" + spec.path, request: payload }));
    }
    expect(operations.size).toBe(25);
  });
  it("preserves immutable code/base unit and the exact concurrency timestamp", () => {
    const spec = formSpec("material-edit", fixtureSource), draft = fixtureDraft("material-edit");
    Object.assign(draft.values, { code: "FORGED", base_unit: "UNIT", expected_updated_at: "forged" });
    const result = buildPayload(spec, draft);
    expect(result.code).toBe("SUP-UI"); expect(result.base_unit).toBe("KG");
    expect(result.expected_updated_at).toBe("2026-11-28T10:11:12.123456Z");
  });
  it("uses the signed-resource version and fixed supplier for revisions", () => {
    const spec = formSpec("po-revise", fixtureSource), draft = fixtureDraft("po-revise");
    draft.values.expected_version = "100"; draft.values.supplier_id = fixtureID;
    expect(buildPayload(spec, draft)).toMatchObject({ expected_version: 7, supplier_id: otherID });
  });
  it("keeps decimal text exact and never invents lot dates or inspection quantities", () => {
    const result = buildPayload(formSpec("po-create", fixtureSource), fixtureDraft("po-create"));
    expect((result.lines as RecordData[])[0].quantity).toBe("0.300000");
    const receipt = formSpec("receipt-create", fixtureSource), inspection = formSpec("receipt-inspect", fixtureSource);
    expect(receipt.initialLines[0].quantity_received).toBe("");
    expect(inspection.initialLines[0].quantity_accepted).toBe("");
    const built = buildPayload(receipt, fixtureDraft("receipt-create"));
    expect((built.lines as RecordData[])[0]).not.toHaveProperty("manufactured_on");
    expect((built.lines as RecordData[])[0]).not.toHaveProperty("expires_on");
  });
  it("sums accepted and rejected quantities exactly, including small decimal fractions", () => {
    const spec = formSpec("receipt-inspect", fixtureSource), draft = fixtureDraft("receipt-inspect");
    expect(buildPayload(spec, draft)).toHaveProperty("lines");
    draft.lines[0].quantity_rejected = "0.199999";
    expect(() => buildPayload(spec, draft)).toThrow("tổng chấp nhận");
    draft.lines[0].quantity_rejected = "0.200000"; draft.lines[0].inspection_notes = "";
    expect(() => buildPayload(spec, draft)).toThrow("ghi chú");
  });
  it("does not remap remaining receipt rows after a row is removed", () => {
    const spec = formSpec("receipt-create", { ...fixtureSource, lines: [{ id: fixtureID }, { id: otherID }] });
    const draft: Draft = { values: { ...spec.initial, warehouse_code: "WH-01" }, lines: [{ purchase_order_line_id: otherID, quantity_received: "1" }] };
    expect((buildPayload(spec, draft).lines as RecordData[])[0].purchase_order_line_id).toBe(otherID);
  });
  it("rejects unsafe money, negative/overprecision quantities, duplicate lines and invalid dates", () => {
    const offer = formSpec("offer-create", fixtureSource), draft = fixtureDraft("offer-create");
    draft.values.unit_price_minor = "9007199254740993"; expect(() => buildPayload(offer, draft)).toThrow("số nguyên chính xác");
    draft.values.unit_price_minor = "12000"; draft.values.effective_from = "2026-02-30"; expect(() => buildPayload(offer, draft)).toThrow("ngày không hợp lệ");
    const po = formSpec("po-create", fixtureSource), poDraft = fixtureDraft("po-create");
    for (const invalid of ["-1", "1e3", "0", "1.1234567"]) { poDraft.lines[0].quantity = invalid; expect(() => buildPayload(po, poDraft)).toThrow(); }
    poDraft.lines[0].quantity = "1"; poDraft.lines.push({ ...poDraft.lines[0] }); expect(() => buildPayload(po, poDraft)).toThrow("trùng");
  });
  it("supports explicit preference clearing and requires a note on supplier change/decline", () => {
    const draft = fixtureDraft("material-preference"); draft.values.supplier_id = "";
    expect(buildPayload(formSpec("material-preference", fixtureSource), draft).supplier_id).toBe("");
    const response = fixtureDraft("po-response"); response.values.note = " ";
    expect(() => buildPayload(formSpec("po-response", fixtureSource), response)).toThrow("nội dung");
  });
});

describe("Workflow action visibility (backend remains authoritative)", () => {
  const permissions = (codes: string[]) => (permission: string) => codes.includes(permission);
  it("never derives permission from ADMIN or grants management to view-only staff", () => {
    expect(availableActions("suppliers", fixtureSource, fixtureID, permissions(["PROCUREMENT_VIEW", "ADMIN"]))).toEqual([]);
  });
  it("separates approval and management and blocks self-approval", () => {
    const row = { ...fixtureSource, status: "PENDING_APPROVAL", created_by: fixtureID };
    expect(availableActions("purchase-orders", row, fixtureID, permissions(["PROCUREMENT_APPROVE"]))).toEqual([]);
    expect(availableActions("purchase-orders", row, otherID, permissions(["PROCUREMENT_APPROVE"]))).toEqual(["po-approve", "po-reject"]);
    expect(availableActions("purchase-orders", row, otherID, permissions(["PROCUREMENT_MANAGE"]))).not.toContain("po-approve");
  });
  it("shows receipt creation/closure on partial PO but not cancellation or draft editing", () => {
    const actions = availableActions("purchase-orders", { ...fixtureSource, status: "PARTIALLY_RECEIVED" }, fixtureID, permissions(["PROCUREMENT_MANAGE", "PROCUREMENT_RECEIVE"]));
    expect(actions).toEqual(["po-close", "receipt-create"]);
  });
  it("does not allow re-inspection or editing an issued PO from stale controls", () => {
    expect(availableActions("goods-receipts", { status: "INSPECTED" }, fixtureID, permissions(["PROCUREMENT_RECEIVE"]))).toEqual([]);
    expect(availableActions("purchase-orders", { status: "SENT" }, fixtureID, permissions(["PROCUREMENT_MANAGE"]))).not.toContain("po-edit");
    expect(availableActions("contracts", { status: "DRAFT" }, fixtureID, permissions(["PROCUREMENT_MANAGE"]))).toEqual(["contract-activate"]);
    expect(availableActions("contracts", { status: "ACTIVE" }, fixtureID, permissions(["PROCUREMENT_MANAGE"]))).toEqual(["contract-terminate"]);
  });
});
