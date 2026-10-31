import type { CreateSupportTicketInput, SupportAttachmentMetadata, SupportPriority, SupportSubjectCode } from "./types";

const SUBJECTS = new Set<SupportSubjectCode>(["PRODUCT_ADVICE", "ORDER_HELP", "DELIVERY_HELP", "PAYMENT_HELP", "RETURN_HELP", "OTHER"]);
const PRIORITIES = new Set<SupportPriority>(["NORMAL", "IMPORTANT", "URGENT"]);
const MEDIA_TYPES = new Set<SupportAttachmentMetadata["mediaType"]>(["image/jpeg", "image/png", "image/webp", "application/pdf"]);
const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/u;
const ID = /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,99}$/u;

type Result<T> = { ok: true; data: T } | { ok: false; errors: Array<{ field: string; message: string }> };
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
const text = (value: unknown) => typeof value === "string" ? value.trim() : "";

export function parseSupportRoute(path: string[]) {
  if (path.length === 1 && path[0] === "context") return { kind: "context" as const };
  if (path.length === 1 && path[0] === "tickets") return { kind: "tickets" as const };
  if (path.length === 2 && path[0] === "tickets" && ID.test(path[1])) return { kind: "ticket" as const, ticketId: path[1] };
  if (path.length === 3 && path[0] === "tickets" && ID.test(path[1]) && path[2] === "messages") return { kind: "messages" as const, ticketId: path[1] };
  return { kind: "invalid" as const };
}

export function validateCreateSupportTicket(value: unknown): Result<CreateSupportTicketInput> {
  const errors: Array<{ field: string; message: string }> = [];
  if (!record(value)) return { ok: false, errors: [{ field: "body", message: "Payload phải là object." }] };
  const subjectCode = text(value.subjectCode) as SupportSubjectCode;
  const title = text(value.title);
  const message = text(value.message);
  const requestedPriority = text(value.requestedPriority) as SupportPriority;
  const linkedOrderId = text(value.linkedOrderId) || undefined;
  const idempotencyKey = text(value.idempotencyKey);
  if (!SUBJECTS.has(subjectCode)) errors.push({ field: "subjectCode", message: "Chủ đề không hợp lệ." });
  if (title.length < 5 || title.length > 160) errors.push({ field: "title", message: "Tiêu đề cần từ 5 đến 160 ký tự." });
  if (message.length < 10 || message.length > 3000) errors.push({ field: "message", message: "Nội dung cần từ 10 đến 3000 ký tự." });
  if (!PRIORITIES.has(requestedPriority)) errors.push({ field: "requestedPriority", message: "Mức độ ưu tiên không hợp lệ." });
  if (linkedOrderId && !ID.test(linkedOrderId)) errors.push({ field: "linkedOrderId", message: "Order không hợp lệ." });
  if (idempotencyKey.length < 16 || idempotencyKey.length > 120) errors.push({ field: "idempotencyKey", message: "Idempotency key không hợp lệ." });

  const attachments: SupportAttachmentMetadata[] = [];
  if (!Array.isArray(value.attachments) || value.attachments.length > 5) errors.push({ field: "attachments", message: "Tối đa 5 tệp đính kèm." });
  else value.attachments.forEach((item, index) => {
    if (!record(item)) { errors.push({ field: `attachments.${index}`, message: "Tệp không hợp lệ." }); return; }
    const mediaType = text(item.mediaType) as SupportAttachmentMetadata["mediaType"];
    const fileName = text(item.fileName);
    const clientReference = text(item.clientReference);
    const sizeBytes = item.sizeBytes;
    if (!clientReference || clientReference.length > 120) errors.push({ field: `attachments.${index}.clientReference`, message: "Mã tệp không hợp lệ." });
    if (!fileName || fileName.length > 180) errors.push({ field: `attachments.${index}.fileName`, message: "Tên tệp không hợp lệ." });
    if (!MEDIA_TYPES.has(mediaType)) errors.push({ field: `attachments.${index}.mediaType`, message: "Loại tệp không hỗ trợ." });
    if (!Number.isInteger(sizeBytes) || Number(sizeBytes) <= 0 || Number(sizeBytes) > 10 * 1024 * 1024) errors.push({ field: `attachments.${index}.sizeBytes`, message: "Tệp phải nhỏ hơn hoặc bằng 10 MiB." });
    attachments.push({ clientReference, fileName, mediaType, sizeBytes: Number(sizeBytes) });
  });

  let contact: CreateSupportTicketInput["contact"];
  if (value.contact !== undefined) {
    if (!record(value.contact)) errors.push({ field: "contact", message: "Thông tin liên hệ không hợp lệ." });
    else {
      const displayName = text(value.contact.displayName);
      const email = text(value.contact.email).toLowerCase();
      const phone = text(value.contact.phone) || undefined;
      if (displayName.length < 2 || displayName.length > 100) errors.push({ field: "contact.displayName", message: "Họ tên không hợp lệ." });
      if (!EMAIL.test(email) || email.length > 254) errors.push({ field: "contact.email", message: "Email không hợp lệ." });
      if (phone && !/^[0-9+() .-]{8,20}$/u.test(phone)) errors.push({ field: "contact.phone", message: "Số điện thoại không hợp lệ." });
      contact = { displayName, email, phone };
    }
  }
  return errors.length ? { ok: false, errors } : { ok: true, data: { subjectCode, title, message, requestedPriority, linkedOrderId, contact, attachments, idempotencyKey } };
}

export function validateSupportMessage(value: unknown): Result<{ message: string; attachments: SupportAttachmentMetadata[]; idempotencyKey: string }> {
  if (!record(value)) return { ok: false, errors: [{ field: "body", message: "Payload phải là object." }] };
  const result = validateCreateSupportTicket({ subjectCode: "OTHER", title: "Bổ sung", message: value.message, requestedPriority: "NORMAL", attachments: value.attachments ?? [], idempotencyKey: value.idempotencyKey });
  if (!result.ok) return { ok: false, errors: result.errors.filter((error) => !["subjectCode", "title", "requestedPriority"].includes(error.field)) };
  return { ok: true, data: { message: result.data.message, attachments: result.data.attachments, idempotencyKey: result.data.idempotencyKey } };
}
