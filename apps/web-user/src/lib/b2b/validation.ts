import type { B2BAttachmentMetadata, B2BPurpose, CreateB2BQuoteRequestInput } from "./types";

const PURPOSES = new Set<B2BPurpose>(["VIP_CUSTOMER_GIFT", "EMPLOYEE_GIFT", "EVENT_GIFT", "PARTNER_GIFT"]);
const MEDIA_TYPES = new Set<B2BAttachmentMetadata["mediaType"]>(["image/png", "image/jpeg", "image/webp", "application/pdf", "image/svg+xml"]);
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const text = (value: unknown) => typeof value === "string" ? value.trim() : "";

export type B2BRoute = { kind: "context" } | { kind: "requests" } | { kind: "quote" | "accept"; quoteId: string } | { kind: "invalid" };

export function parseB2BRoute(path: string[]): B2BRoute {
  if (path.length === 1 && path[0] === "context") return { kind: "context" };
  if (path.length === 1 && path[0] === "quote-requests") return { kind: "requests" };
  if (path.length === 2 && path[0] === "quotes" && path[1]) return { kind: "quote", quoteId: path[1] };
  if (path.length === 3 && path[0] === "quotes" && path[1] && path[2] === "accept") return { kind: "accept", quoteId: path[1] };
  return { kind: "invalid" };
}

export function validateCreateB2BQuoteRequest(value: unknown): { ok: true; data: CreateB2BQuoteRequestInput } | { ok: false; errors: Array<{ field: string; message: string }> } {
  const errors: Array<{ field: string; message: string }> = [];
  if (!record(value)) return { ok: false, errors: [{ field: "body", message: "Dữ liệu yêu cầu không hợp lệ." }] };
  const organizationId = text(value.organizationId);
  const purpose = text(value.purpose) as B2BPurpose;
  const requestedDeliveryDate = text(value.requestedDeliveryDate);
  const deliveryProvinceCode = text(value.deliveryProvinceCode);
  const deliveryProvinceName = text(value.deliveryProvinceName);
  const notes = text(value.notes);
  const idempotencyKey = text(value.idempotencyKey);
  if (!organizationId) errors.push({ field: "organizationId", message: "Thiếu doanh nghiệp gửi yêu cầu." });
  if (!PURPOSES.has(purpose)) errors.push({ field: "purpose", message: "Mục đích quà tặng không hợp lệ." });
  const deliveryTime = Date.parse(requestedDeliveryDate);
  if (!requestedDeliveryDate || Number.isNaN(deliveryTime)) errors.push({ field: "requestedDeliveryDate", message: "Ngày giao dự kiến không hợp lệ." });
  else if (deliveryTime < Date.now() + 6 * 24 * 60 * 60 * 1000) errors.push({ field: "requestedDeliveryDate", message: "Ngày giao dự kiến cần cách hiện tại ít nhất 7 ngày." });
  if (!deliveryProvinceCode || !deliveryProvinceName) errors.push({ field: "deliveryProvinceCode", message: "Hãy chọn Tỉnh/Thành phố giao hàng." });
  const rawItems = Array.isArray(value.items) ? value.items : [];
  const items = rawItems.flatMap((item) => record(item) && text(item.skuId) && Number.isInteger(item.quantity) && Number(item.quantity) > 0 ? [{ skuId: text(item.skuId), quantity: Number(item.quantity) }] : []);
  if (!items.length || items.length !== rawItems.length) errors.push({ field: "items", message: "Hãy chọn sản phẩm và số lượng hợp lệ." });
  if (new Set(items.map((item) => item.skuId)).size !== items.length) errors.push({ field: "items", message: "Mỗi SKU chỉ được xuất hiện một lần." });
  if (!record(value.branding)) errors.push({ field: "branding", message: "Thông tin tùy biến không hợp lệ." });
  const branding = record(value.branding) ? value.branding : {};
  const rawAttachments = Array.isArray(branding.attachments) ? branding.attachments : [];
  const attachments = rawAttachments.flatMap((attachment) => {
    if (!record(attachment)) return [];
    const mediaType = text(attachment.mediaType) as B2BAttachmentMetadata["mediaType"];
    const sizeBytes = Number(attachment.sizeBytes);
    return text(attachment.clientReference) && text(attachment.fileName) && MEDIA_TYPES.has(mediaType) && Number.isFinite(sizeBytes) && sizeBytes > 0 && sizeBytes <= 25 * 1024 * 1024
      ? [{ clientReference: text(attachment.clientReference), fileName: text(attachment.fileName), mediaType, sizeBytes }]
      : [];
  });
  if (attachments.length !== rawAttachments.length || attachments.length > 3) errors.push({ field: "branding.attachments", message: "Tệp nhận diện không hợp lệ hoặc vượt giới hạn." });
  if (notes.length > 2000) errors.push({ field: "notes", message: "Ghi chú không được vượt quá 2.000 ký tự." });
  if (idempotencyKey.length < 16 || idempotencyKey.length > 160) errors.push({ field: "idempotencyKey", message: "Mã chống gửi trùng không hợp lệ." });
  if (typeof value.invoiceRequested !== "boolean" || typeof value.sampleRequested !== "boolean") errors.push({ field: "confirmation", message: "Thiếu lựa chọn hóa đơn hoặc hộp mẫu." });
  if (errors.length) return { ok: false, errors };
  return { ok: true, data: { organizationId, purpose, requestedDeliveryDate, deliveryProvinceCode, deliveryProvinceName, items, branding: { engraveLogo: Boolean(branding.engraveLogo), customSleeve: Boolean(branding.customSleeve), greetingCard: Boolean(branding.greetingCard), ribbon: Boolean(branding.ribbon), attachments }, notes: notes || undefined, invoiceRequested: value.invoiceRequested as boolean, sampleRequested: value.sampleRequested as boolean, idempotencyKey } };
}
