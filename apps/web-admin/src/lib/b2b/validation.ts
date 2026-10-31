import type { AdminB2BActionInput } from "./types";

const ACTIONS = new Set<AdminB2BActionInput["action"]>(["ASSIGN", "REQUEST_INFO", "CREATE_DRAFT", "ISSUE", "WITHDRAW", "REJECT", "CONVERT_ORDER"]);
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const text = (value: unknown) => typeof value === "string" ? value.trim() : "";

export function parseAdminB2BPath(path: string[]) {
  if (path.length === 1 && path[0] === "quote-requests") return { kind: "list" as const };
  if (path.length === 2 && path[0] === "quote-requests" && path[1]) return { kind: "detail" as const, requestId: path[1] };
  if (path.length === 3 && path[0] === "quote-requests" && path[1] && path[2] === "files") return { kind: "files" as const, requestId: path[1] };
  if (path.length === 3 && path[0] === "quote-requests" && path[1] && path[2] === "versions") return { kind: "versions" as const, requestId: path[1] };
  if (path.length === 3 && path[0] === "quote-requests" && path[1] && path[2] === "actions") return { kind: "action" as const, requestId: path[1] };
  return { kind: "invalid" as const };
}

export function validateAdminB2BAction(value: unknown): { ok: true; data: AdminB2BActionInput } | { ok: false; errors: Array<{ field: string; message: string }> } {
  if (!record(value)) return { ok: false, errors: [{ field: "body", message: "Dữ liệu thao tác không hợp lệ." }] };
  const action = text(value.action) as AdminB2BActionInput["action"];
  const reason = text(value.reason);
  const assigneeId = text(value.assigneeId);
  const idempotencyKey = text(value.idempotencyKey);
  const expectedRevision = Number(value.expectedRevision);
  const errors: Array<{ field: string; message: string }> = [];
  if (!ACTIONS.has(action)) errors.push({ field: "action", message: "Thao tác không được hỗ trợ." });
  if (idempotencyKey.length < 16 || idempotencyKey.length > 160) errors.push({ field: "idempotencyKey", message: "Mã chống gửi trùng không hợp lệ." });
  if (!Number.isInteger(expectedRevision) || expectedRevision < 1) errors.push({ field: "expectedRevision", message: "Revision dữ liệu không hợp lệ; hãy tải lại yêu cầu." });
  if (action === "ASSIGN" && !assigneeId) errors.push({ field: "assigneeId", message: "Hãy chọn nhân viên phụ trách." });
  if ((action === "REQUEST_INFO" || action === "WITHDRAW" || action === "REJECT") && reason.length < 10) errors.push({ field: "reason", message: "Lý do cần ít nhất 10 ký tự." });
  const invoiceSnapshotConfirmed = value.invoiceSnapshotConfirmed === true;
  const availabilityCheckAcknowledged = value.availabilityCheckAcknowledged === true;
  if (action === "CONVERT_ORDER" && !invoiceSnapshotConfirmed) errors.push({ field: "invoiceSnapshotConfirmed", message: "Phải xác nhận snapshot thông tin hóa đơn." });
  if (action === "CONVERT_ORDER" && !availabilityCheckAcknowledged) errors.push({ field: "availabilityCheckAcknowledged", message: "Phải xác nhận đã kiểm tra lại khả năng đáp ứng." });
  let quote: AdminB2BActionInput["quote"];
  if (action === "CREATE_DRAFT") {
    if (!record(value.quote)) errors.push({ field: "quote", message: "Thiếu thông tin báo giá nháp." });
    else {
      const expiresAt = text(value.quote.expiresAt);
      const terms = text(value.quote.terms);
      const discountPercent = Number(value.quote.discountPercent);
      const customizationVnd = Number(value.quote.customizationVnd);
      const shippingVnd = Number(value.quote.shippingVnd);
      const vatPercent = Number(value.quote.vatPercent);
      if (![discountPercent, customizationVnd, shippingVnd, vatPercent].every(Number.isFinite) || discountPercent < 0 || discountPercent > 100 || customizationVnd < 0 || shippingVnd < 0 || vatPercent < 0 || vatPercent > 100) errors.push({ field: "quote", message: "Giá, chiết khấu hoặc thuế không hợp lệ." });
      if (!expiresAt || Number.isNaN(Date.parse(expiresAt))) errors.push({ field: "quote.expiresAt", message: "Ngày hết hiệu lực không hợp lệ." });
      if (terms.length < 10) errors.push({ field: "quote.terms", message: "Điều khoản cần ít nhất 10 ký tự." });
      quote = { discountPercent, customizationVnd, shippingVnd, vatPercent, expiresAt, terms };
    }
  }
  if (errors.length) return { ok: false, errors };
  return { ok: true, data: { action, expectedRevision, reason: reason || undefined, assigneeId: assigneeId || undefined, quote, invoiceSnapshotConfirmed: action === "CONVERT_ORDER" ? true : undefined, availabilityCheckAcknowledged: action === "CONVERT_ORDER" ? true : undefined, idempotencyKey } };
}
