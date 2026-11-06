import type { InventoryAdjustmentInput, InventoryIssueInput, InventoryReceiptInput } from "./types";

const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const text = (value: unknown) => typeof value === "string" ? value.trim() : "";

export function parseAdminInventoryPath(path: string[]) {
  if (path.length === 1 && path[0] === "skus") return { kind: "sku-list" as const };
  if (path.length === 2 && path[0] === "skus" && path[1]) return { kind: "sku-detail" as const, skuId: path[1] };
  if (path.length === 1 && path[0] === "workbench") return { kind: "workbench-context" as const };
  if (path.length === 2 && path[0] === "workbench" && path[1] === "receipts") return { kind: "receipt-create" as const };
  if (path.length === 2 && path[0] === "workbench" && path[1] === "issues") return { kind: "issue-create" as const };
  if (path.length === 2 && path[0] === "workbench" && path[1] === "adjustments") return { kind: "adjustment-create" as const };
  return { kind: "invalid" as const };
}

type ValidationError = { field: string; message: string };

function common(value: unknown) {
  if (!record(value)) return { source: undefined, errors: [{ field: "body", message: "Dữ liệu thao tác không hợp lệ." }] as ValidationError[] };
  const skuId = text(value.skuId);
  const expectedRevision = Number(value.expectedRevision);
  const idempotencyKey = text(value.idempotencyKey);
  const errors: ValidationError[] = [];
  if (!skuId) errors.push({ field: "skuId", message: "Hãy chọn SKU." });
  if (!Number.isInteger(expectedRevision) || expectedRevision < 1) errors.push({ field: "expectedRevision", message: "Revision không hợp lệ; hãy tải lại Workbench." });
  if (idempotencyKey.length < 16 || idempotencyKey.length > 160) errors.push({ field: "idempotencyKey", message: "Mã chống gửi trùng không hợp lệ." });
  return { source: value, skuId, expectedRevision, idempotencyKey, errors };
}

function positiveInteger(value: unknown, field: string, errors: ValidationError[]) {
  const quantity = Number(value);
  if (!Number.isInteger(quantity) || quantity < 1) errors.push({ field, message: "Số lượng phải là số nguyên dương." });
  return quantity;
}

function reference(value: unknown, field: "reference" | "sourceReference", errors: ValidationError[]) {
  const result = text(value);
  if (result.length < 3 || result.length > 100) errors.push({ field, message: "Tham chiếu phải từ 3 đến 100 ký tự." });
  return result;
}

export function validateInventoryReceipt(value: unknown): { ok: true; data: InventoryReceiptInput } | { ok: false; errors: ValidationError[] } {
  const parsed = common(value);
  if (!parsed.source) return { ok: false, errors: parsed.errors };
  const batchMode = text(parsed.source.batchMode) as InventoryReceiptInput["batchMode"];
  const batchId = text(parsed.source.batchId);
  const batchCode = text(parsed.source.batchCode);
  const manufacturingDate = text(parsed.source.manufacturingDate);
  const expiresAt = text(parsed.source.expiresAt);
  const quantity = positiveInteger(parsed.source.quantity, "quantity", parsed.errors);
  const sourceReference = reference(parsed.source.sourceReference, "sourceReference", parsed.errors);
  if (batchMode !== "EXISTING" && batchMode !== "NEW") parsed.errors.push({ field: "batchMode", message: "Cách ghi nhận Batch không hợp lệ." });
  if (batchMode === "EXISTING" && !batchId) parsed.errors.push({ field: "batchId", message: "Hãy chọn Batch nhận hàng." });
  if (batchMode === "NEW") {
    if (batchCode.length < 2 || batchCode.length > 80) parsed.errors.push({ field: "batchCode", message: "Mã Batch phải từ 2 đến 80 ký tự." });
    const manufactured = Date.parse(manufacturingDate);
    const expiry = Date.parse(expiresAt);
    if (Number.isNaN(manufactured)) parsed.errors.push({ field: "manufacturingDate", message: "Ngày sản xuất không hợp lệ." });
    if (Number.isNaN(expiry)) parsed.errors.push({ field: "expiresAt", message: "Hạn sử dụng không hợp lệ." });
    if (!Number.isNaN(manufactured) && !Number.isNaN(expiry) && expiry <= manufactured) parsed.errors.push({ field: "expiresAt", message: "Hạn sử dụng phải sau ngày sản xuất." });
  }
  if (parsed.errors.length) return { ok: false, errors: parsed.errors };
  return { ok: true, data: { skuId: parsed.skuId!, batchMode, batchId: batchId || undefined, batchCode: batchCode || undefined, manufacturingDate: manufacturingDate || undefined, expiresAt: expiresAt || undefined, quantity, sourceReference, expectedRevision: parsed.expectedRevision!, idempotencyKey: parsed.idempotencyKey! } };
}

export function validateInventoryIssue(value: unknown): { ok: true; data: InventoryIssueInput } | { ok: false; errors: ValidationError[] } {
  const parsed = common(value);
  if (!parsed.source) return { ok: false, errors: parsed.errors };
  const batchId = text(parsed.source.batchId);
  const purpose = text(parsed.source.purpose) as InventoryIssueInput["purpose"];
  const quantity = positiveInteger(parsed.source.quantity, "quantity", parsed.errors);
  const resultReference = reference(parsed.source.reference, "reference", parsed.errors);
  const reason = text(parsed.source.reason);
  if (!batchId) parsed.errors.push({ field: "batchId", message: "Hãy chọn Batch xuất kho." });
  if (purpose !== "ORDER" && purpose !== "APPROVED_NON_ORDER") parsed.errors.push({ field: "purpose", message: "Mục đích xuất kho không hợp lệ." });
  if (purpose === "APPROVED_NON_ORDER" && reason.length < 10) parsed.errors.push({ field: "reason", message: "Xuất ngoài Order cần lý do ít nhất 10 ký tự." });
  if (reason.length > 500) parsed.errors.push({ field: "reason", message: "Lý do không được vượt quá 500 ký tự." });
  if (parsed.errors.length) return { ok: false, errors: parsed.errors };
  return { ok: true, data: { skuId: parsed.skuId!, batchId, purpose, quantity, reference: resultReference, reason: reason || undefined, expectedRevision: parsed.expectedRevision!, idempotencyKey: parsed.idempotencyKey! } };
}

export function validateInventoryAdjustment(value: unknown): { ok: true; data: InventoryAdjustmentInput } | { ok: false; errors: ValidationError[] } {
  const parsed = common(value);
  if (!parsed.source) return { ok: false, errors: parsed.errors };
  const batchId = text(parsed.source.batchId);
  const kind = text(parsed.source.kind) as InventoryAdjustmentInput["kind"];
  const resultReference = reference(parsed.source.reference, "reference", parsed.errors);
  const reason = text(parsed.source.reason);
  let quantity: number | undefined;
  let countedQuantity: number | undefined;
  if (!batchId) parsed.errors.push({ field: "batchId", message: "Hãy chọn Batch cần ghi nhận." });
  if (!(["DAMAGE", "LOSS", "COUNT"] as const).includes(kind)) parsed.errors.push({ field: "kind", message: "Loại điều chỉnh không hợp lệ." });
  if (kind === "COUNT") {
    countedQuantity = Number(parsed.source.countedQuantity);
    if (!Number.isInteger(countedQuantity) || countedQuantity < 0) parsed.errors.push({ field: "countedQuantity", message: "Số lượng kiểm đếm phải là số nguyên không âm." });
  } else quantity = positiveInteger(parsed.source.quantity, "quantity", parsed.errors);
  if (reason.length < 10 || reason.length > 500) parsed.errors.push({ field: "reason", message: "Lý do phải từ 10 đến 500 ký tự." });
  if (parsed.errors.length) return { ok: false, errors: parsed.errors };
  return { ok: true, data: { skuId: parsed.skuId!, batchId, kind, quantity, countedQuantity, reference: resultReference, reason, expectedRevision: parsed.expectedRevision!, idempotencyKey: parsed.idempotencyKey! } };
}
