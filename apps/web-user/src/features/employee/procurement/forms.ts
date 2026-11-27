export type RecordData = Record<string, unknown>;
export type ResourceKind = "suppliers" | "materials" | "purchase-orders" | "goods-receipts" | "contracts";
export type Field = { name: string; label: string; type?: "text" | "textarea" | "uuid" | "date" | "integer" | "quantity" | "email" | "boolean"; required?: boolean; locked?: boolean; options?: string[]; positive?: boolean; max?: number; maxLength?: number; allowEmpty?: boolean; hint?: string };
export type FormSpec = { action: string; title: string; method: "POST" | "PUT" | "PATCH"; path: string; fields: Field[]; lineFields?: Field[]; fixedLines?: boolean; initial: Record<string, string>; initialLines: Array<Record<string, string>>; source: RecordData };
export type Draft = { values: Record<string, string>; lines: Array<Record<string, string>> };
export const UUID_PATTERN = /^(?!00000000-0000-0000-0000-000000000000)[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/u;
export const rowText = (v: unknown): string => typeof v === "string" || typeof v === "number" || typeof v === "boolean" ? String(v) : "";
export const records = (v: unknown): RecordData[] => Array.isArray(v) ? v.filter((r): r is RecordData => Boolean(r) && typeof r === "object" && !Array.isArray(r)) : [];
const f = (name: string, label: string, rest: Omit<Field, "name" | "label"> = {}): Field => ({ name, label, ...rest });
const id = (name: string, label: string, locked = false) => f(name, label, { type: "uuid", required: true, locked });
const reason = f("reason", "Lý do", { type: "textarea", required: true, maxLength: 2000 });
const version = f("expected_version", "Phiên bản đang thao tác", { type: "integer", required: true, locked: true, positive: true });
const timestamp = f("expected_updated_at", "Mốc cập nhật đang thao tác", { required: true, locked: true, hint: "Giữ nguyên mốc chính xác từ máy chủ để tránh ghi đè dữ liệu mới." });
const notes = f("notes", "Ghi chú", { type: "textarea", maxLength: 4000 });
const deliveryDate = f("expected_delivery_date", "Ngày dự kiến nhận", { type: "date" });
const poLines = [id("material_id", "ID nguyên liệu"), f("quantity", "Số lượng đặt", { type: "quantity", required: true, positive: true })];
const supplierFields = [f("code", "Mã nhà cung cấp", { required: true, maxLength: 80 }), f("name", "Tên nhà cung cấp", { required: true, maxLength: 240 }), f("supplier_type", "Loại nhà cung cấp", { required: true, options: ["FARMER", "COOPERATIVE", "OCOP_BUSINESS", "COMPANY"] }), f("contact_name", "Người liên hệ", { maxLength: 160 }), f("phone", "Số điện thoại", { maxLength: 40 }), f("email", "Email", { type: "email", maxLength: 254 }), f("address", "Địa chỉ", { type: "textarea", maxLength: 4000 }), f("legal_identifier", "Mã số pháp lý / mã số thuế", { maxLength: 120 })];
const materialFields = [f("code", "Mã nguyên liệu", { required: true, maxLength: 80 }), f("name", "Tên nguyên liệu", { required: true, maxLength: 240 }), f("base_unit", "Đơn vị cơ sở", { required: true, maxLength: 24, hint: "Đơn vị này không tự tạo cấu hình quy đổi trong Inventory." }), f("description", "Mô tả", { type: "textarea", maxLength: 4000 })];

/** Shapes intentionally follow the existing 25 write operations, not a JSON console. */
export function formSpec(action: string, source: RecordData = {}): FormSpec {
  const resourceID = rowText(source.id), supplierID = rowText(source.supplier_id);
  const spec: FormSpec = { action, title: "", method: "POST", path: "", fields: [], initial: {}, initialLines: [], source };
  const set = (title: string, path: string, fields: Field[], method: FormSpec["method"] = "POST") => Object.assign(spec, { title, path, fields, method });
  switch (action) {
    case "supplier-create": set("Tạo nhà cung cấp", "suppliers", supplierFields); break;
    case "supplier-edit": set("Sửa nhà cung cấp", `suppliers/${resourceID}`, [...supplierFields.map((field) => ({ ...field, locked: field.name === "code" })), timestamp], "PUT"); break;
    case "supplier-status": set("Đổi trạng thái nhà cung cấp", `suppliers/${resourceID}/status`, [f("status", "Trạng thái", { required: true, options: ["ACTIVE", "SUSPENDED"] }), timestamp], "PATCH"); break;
    case "material-create": set("Tạo nguyên liệu", "materials", materialFields); break;
    case "material-edit": set("Sửa nguyên liệu", `materials/${resourceID}`, [...materialFields.map((field) => ({ ...field, locked: ["code", "base_unit"].includes(field.name) })), timestamp], "PUT"); break;
    case "material-status": set("Đổi trạng thái nguyên liệu", `materials/${resourceID}/status`, [f("status", "Trạng thái", { required: true, options: ["ACTIVE", "INACTIVE"] }), timestamp], "PATCH"); break;
    case "material-preference": set("Chọn / bỏ nhà cung cấp ưu tiên", `materials/${resourceID}/preferred-supplier`, [f("supplier_id", "ID nhà cung cấp ưu tiên", { type: "uuid", required: true, allowEmpty: true, hint: "Để trống để bỏ ưu tiên; nhà cung cấp được chọn phải có báo giá phù hợp." }), timestamp], "PUT"); break;
    case "offer-create": set("Tạo phiên bản bảng giá", `suppliers/${resourceID}/offers`, [id("material_id", "ID nguyên liệu"), f("supplier_sku", "Mã hàng của nhà cung cấp", { maxLength: 100 }), f("unit", "Đơn vị báo giá", { required: true, maxLength: 24 }), f("unit_price_minor", "Đơn giá (đơn vị tiền nhỏ nhất)", { type: "integer", required: true, positive: true }), f("currency", "Mã tiền tệ", { required: true, maxLength: 3 }), f("minimum_quantity", "Số lượng đặt tối thiểu", { type: "quantity", required: true, positive: true }), f("lead_time_days", "Thời gian cung ứng (ngày)", { type: "integer", required: true, max: 2147483647 }), f("effective_from", "Hiệu lực từ ngày (UTC)", { type: "date", required: true }), f("effective_to", "Hiệu lực hết ngày (UTC)", { type: "date", hint: "Ngày cuối được tính bao gồm; để trống nếu không giới hạn." })]); break;
    case "offer-status": set("Bật / tắt phiên bản bảng giá", `suppliers/${supplierID}/offers/${resourceID}/status`, [f("active", "Đang hoạt động", { type: "boolean", required: true }), timestamp], "PATCH"); break;
    case "contract-create": set("Tạo phiên bản hợp đồng", `suppliers/${resourceID}/contracts`, [f("contract_number", "Số hợp đồng", { required: true, maxLength: 100 }), f("scope", "Phạm vi hợp đồng", { required: true, maxLength: 240 }), f("payment_terms", "Điều khoản thanh toán", { type: "textarea", required: true, maxLength: 4000 }), f("effective_from", "Hiệu lực từ ngày (UTC)", { type: "date", required: true }), f("effective_to", "Hiệu lực hết ngày (UTC)", { type: "date" })]); break;
    case "contract-activate": set("Kích hoạt hợp đồng", `contracts/${resourceID}/activate`, [timestamp]); break;
    case "contract-terminate": set("Chấm dứt hợp đồng", `contracts/${resourceID}/terminate`, [timestamp, reason]); break;
    case "po-create": set("Tạo đơn mua hàng", "purchase-orders", [id("supplier_id", "ID nhà cung cấp"), deliveryDate, notes]); spec.lineFields = poLines; break;
    case "po-edit": case "po-revise":
      set(action === "po-edit" ? "Sửa bản nháp PO" : "Tạo revision PO mới", `purchase-orders/${resourceID}${action === "po-revise" ? "/revisions" : ""}`, [version, id("supplier_id", "ID nhà cung cấp", action === "po-revise"), deliveryDate, notes, ...(action === "po-revise" ? [reason] : [])], action === "po-edit" ? "PUT" : "POST"); spec.lineFields = poLines; break;
    case "po-submit": case "po-approve": case "po-reject": case "po-send": case "po-cancel": case "po-close": {
      const command = action.slice(3);
      const labels: Record<string, string> = { submit: "Trình duyệt PO", approve: "Duyệt PO", reject: "Từ chối PO", send: "Xác nhận đã gửi PO", cancel: "Hủy PO", close: "Đóng phần còn lại của PO" };
      set(labels[command], `purchase-orders/${resourceID}/${command}`, [version, ...(["reject", "cancel", "close"].includes(command) ? [reason] : [])]); break;
    }
    case "po-response": set("Ghi nhận phản hồi nhà cung cấp", `purchase-orders/${resourceID}/supplier-response`, [version, f("response", "Phản hồi", { required: true, options: ["ACKNOWLEDGED", "CHANGE_REQUESTED", "DECLINED"] }), f("note", "Nội dung phản hồi", { type: "textarea", maxLength: 4000, hint: "Bắt buộc khi đề nghị sửa hoặc từ chối." })]); break;
    case "receipt-create": set("Tạo phiếu nhận hàng", "goods-receipts", [id("purchase_order_id", "ID đơn mua hàng", Boolean(resourceID)), f("warehouse_code", "Mã kho thực nhận", { required: true, maxLength: 128 }), notes]);
      spec.lineFields = [id("purchase_order_line_id", "ID dòng PO"), f("quantity_received", "Số lượng thực nhận", { type: "quantity", required: true, positive: true }), f("supplier_lot_code", "Mã lô nhà cung cấp", { maxLength: 240 }), f("manufactured_on", "Ngày sản xuất đã xác minh", { type: "date" }), f("expires_on", "Ngày hết hạn đã xác minh", { type: "date" })]; break;
    case "receipt-inspect": set("Kiểm nhận nguyên liệu", `goods-receipts/${resourceID}/inspect`, []);
      spec.fixedLines = true; spec.lineFields = [id("goods_receipt_line_id", "ID dòng phiếu nhận", true), f("quantity_accepted", "Số lượng chấp nhận", { type: "quantity", required: true }), f("quantity_rejected", "Số lượng từ chối", { type: "quantity", required: true }), f("humidity_percent", "Độ ẩm (%)", { type: "quantity", max: 100 }), f("inspection_notes", "Ghi chú kiểm nhận", { type: "textarea", maxLength: 4000, hint: "Bắt buộc nếu từ chối hoặc nhận dư." })]; break;
    case "variance-resolve": set("Xử lý chênh lệch nhận hàng", `goods-receipts/${rowText(source.receipt_id)}/variances/${resourceID}/resolve`, [version, f("decision", "Quyết định", { required: true, options: ["ACCEPT_VARIANCE", "REJECT_VARIANCE"] }), f("note", "Lý do xử lý", { type: "textarea", required: true, maxLength: 4000 })]); break;
    default: throw new Error("Thao tác không được hỗ trợ.");
  }
  for (const field of spec.fields) {
    const raw = field.name === "expected_version" ? source.version : field.name === "expected_updated_at" ? source.updated_at : source[field.name];
    spec.initial[field.name] = field.type === "date" ? rowText(raw).slice(0, 10) : rowText(raw);
  }
  if (action === "material-preference") spec.initial.supplier_id = rowText(source.preferred_supplier_id);
  if (action === "offer-create") { spec.initial.currency = "VND"; spec.initial.lead_time_days = "0"; }
  if (action === "receipt-create" && resourceID) spec.initial.purchase_order_id = resourceID;
  if (spec.lineFields) {
    spec.initialLines = records(source.lines).map((line) => Object.fromEntries(spec.lineFields!.map((field) => [field.name,
      field.name === "purchase_order_line_id" || field.name === "goods_receipt_line_id" ? rowText(line.id)
        : action === "receipt-create" || action === "receipt-inspect" ? "" : rowText(line[field.name])])));
    if (!spec.initialLines.length && !spec.fixedLines) spec.initialLines = [{}];
  }
  // Dates on a newly created contract/offer are entered by the user, never copied
  // from a supplier timestamp or synthesized as verified business evidence.
  if (["contract-create", "offer-create"].includes(action)) for (const field of spec.fields) if (field.type === "date") spec.initial[field.name] = "";
  return spec;
}

function scalar(field: Field, input: string | undefined): string | number | boolean | undefined {
  const value = input?.trim() ?? "";
  if (!value && field.allowEmpty) return "";
  if (!value) { if (field.required) throw new Error(`${field.label}: không được để trống.`); return field.type === "date" || field.type === "integer" || field.type === "quantity" ? undefined : ""; }
  if (value.length > (field.maxLength ?? 4000)) throw new Error(`${field.label}: vượt độ dài cho phép.`);
  if (field.options && !field.options.includes(value)) throw new Error(`${field.label}: lựa chọn không hợp lệ.`);
  if (field.type === "uuid" && !UUID_PATTERN.test(value)) throw new Error(`${field.label}: UUID không hợp lệ.`);
  if (field.type === "email" && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/u.test(value)) throw new Error(`${field.label}: email không hợp lệ.`);
  if (field.type === "date") {
    if (!/^\d{4}-\d{2}-\d{2}$/u.test(value) || Number.isNaN(Date.parse(value + "T00:00:00Z")) || new Date(value + "T00:00:00Z").toISOString().slice(0, 10) !== value) throw new Error(`${field.label}: ngày không hợp lệ.`);
  }
  if (field.type === "integer") {
    if (!/^\d+$/u.test(value) || !Number.isSafeInteger(Number(value)) || Number(value) < (field.positive ? 1 : 0) || Number(value) > (field.max ?? Number.MAX_SAFE_INTEGER)) throw new Error(`${field.label}: phải là số nguyên chính xác trong giới hạn cho phép.`);
    return Number(value);
  }
  if (field.type === "quantity") {
    if (!/^\d+(?:\.\d{1,6})?$/u.test(value) || BigInt(value.split(".")[0]) >= BigInt("1000000000000") || field.positive && decimal(value) === BigInt(0) || field.max !== undefined && decimal(value) > BigInt(field.max) * BigInt(1000000)) throw new Error(`${field.label}: số thập phân không hợp lệ (tối đa 6 chữ số sau dấu chấm).`);
  }
  if (field.type === "boolean") { if (!["true", "false"].includes(value)) throw new Error(`${field.label}: lựa chọn không hợp lệ.`); return value === "true"; }
  return value;
}
function decimal(value: string): bigint { const [whole, fraction = ""] = value.split("."); return BigInt(whole) * BigInt(1000000) + BigInt(fraction.padEnd(6, "0")); }

export function buildPayload(spec: FormSpec, draft: Draft): RecordData {
  const result: RecordData = {};
  for (const field of spec.fields) {
    const value = scalar(field, field.locked ? spec.initial[field.name] : draft.values[field.name]);
    if (value !== undefined) result[field.name] = value;
  }
  if (spec.lineFields) {
    if (!draft.lines.length || draft.lines.length > 100 || spec.fixedLines && draft.lines.length !== spec.initialLines.length) throw new Error("Cần từ 1 đến 100 dòng, giữ đủ các dòng khi kiểm nhận.");
    result.lines = draft.lines.map((line, index) => {
      const row: RecordData = {};
      for (const field of spec.lineFields!) {
        const value = scalar(field, field.locked ? spec.initialLines[index]?.[field.name] : line[field.name]);
        if (value !== undefined) row[field.name] = value;
      }
      if (row.manufactured_on && row.expires_on && rowText(row.expires_on) < rowText(row.manufactured_on)) throw new Error("Hạn sử dụng không được trước ngày sản xuất.");
      return row;
    });
    const key = spec.lineFields[0].name;
    if (new Set((result.lines as RecordData[]).map((line) => line[key])).size !== draft.lines.length) throw new Error("Không được trùng nguyên liệu hoặc dòng chứng từ.");
  }
  if (result.currency && !/^[A-Za-z]{3}$/u.test(rowText(result.currency))) throw new Error("Mã tiền tệ phải có 3 chữ cái.");
  if (result.effective_from && result.effective_to && rowText(result.effective_to) < rowText(result.effective_from)) throw new Error("Ngày kết thúc không được trước ngày bắt đầu.");
  if (spec.action === "po-response" && result.response !== "ACKNOWLEDGED" && !rowText(result.note).trim()) throw new Error("Cần ghi nội dung khi đề nghị sửa hoặc từ chối PO.");
  if (spec.action === "receipt-inspect") {
    const original = records(spec.source.lines);
    for (const [index, line] of (result.lines as RecordData[]).entries()) {
      if (decimal(rowText(line.quantity_accepted)) + decimal(rowText(line.quantity_rejected)) !== decimal(rowText(original[index].quantity_received))) throw new Error(`Dòng ${index + 1}: tổng chấp nhận và từ chối phải bằng số lượng thực nhận.`);
      if ((decimal(rowText(line.quantity_rejected)) > BigInt(0) || decimal(rowText(original[index].quantity_received)) > decimal(rowText(original[index].expected_open_quantity))) && !rowText(line.inspection_notes).trim()) throw new Error(`Dòng ${index + 1}: cần ghi chú khi từ chối hoặc nhận dư.`);
    }
  }
  return result;
}

/** UI hints only; backend independently checks signed scope and workflow. */
export function availableActions(kind: ResourceKind, row: RecordData, actor: string, has: (permission: string) => boolean): string[] {
  const actions: string[] = [], status = rowText(row.status);
  const manage = has("PROCUREMENT_MANAGE");
  if (kind === "suppliers" && manage) actions.push("supplier-edit", "supplier-status", ...(status === "ACTIVE" ? ["offer-create", "contract-create"] : []));
  if (kind === "materials" && manage) actions.push("material-edit", "material-status", "material-preference");
  if (kind === "contracts" && manage) { if (status === "DRAFT") actions.push("contract-activate"); if (status === "ACTIVE") actions.push("contract-terminate"); }
  if (kind === "purchase-orders") {
    if (manage) {
      if (status === "DRAFT") actions.push("po-edit", "po-submit");
      if (status === "APPROVED") actions.push("po-send");
      if (status === "SENT") actions.push("po-response");
      if (status === "CHANGE_REQUESTED") actions.push("po-revise");
      if (["DRAFT", "PENDING_APPROVAL", "APPROVED", "REJECTED"].includes(status)) actions.push("po-cancel");
      if (["SENT", "ACKNOWLEDGED", "PARTIALLY_RECEIVED"].includes(status)) actions.push("po-close");
    }
    if (has("PROCUREMENT_APPROVE") && status === "PENDING_APPROVAL" && row.created_by !== actor) actions.push("po-approve", "po-reject");
    if (has("PROCUREMENT_RECEIVE") && ["SENT", "ACKNOWLEDGED", "PARTIALLY_RECEIVED"].includes(status)) actions.push("receipt-create");
  }
  if (kind === "goods-receipts" && has("PROCUREMENT_RECEIVE") && status === "DRAFT") actions.push("receipt-inspect");
  return actions;
}
