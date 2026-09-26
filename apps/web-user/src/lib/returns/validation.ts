import type { CreateReturnCaseInput, EvidenceMetadata, ReturnReason, ReturnResolution } from "./types";

const ID_PATTERN = /^[A-Za-z0-9-]{8,100}$/u;
const SCENARIO_PATTERN = /^[a-z0-9-]+$/u;
const CREATE_KEY_PATTERN = /^return-case-[0-9a-f-]{36}$/u;
const SUPPLEMENT_KEY_PATTERN = /^return-supplement-[0-9a-f-]{36}$/u;
const REASONS = new Set<ReturnReason>(["DAMAGED_IN_TRANSIT", "WRONG_ITEM", "QUALITY_ISSUE", "OTHER"]);
const RESOLUTIONS = new Set<ReturnResolution>(["REPLACEMENT", "ORIGINAL_PAYMENT_REFUND", "LOYALTY_CREDIT"]);
const MEDIA_TYPES = new Set<EvidenceMetadata["mediaType"]>(["image/jpeg", "image/png", "image/webp", "video/mp4"]);
const MAX_FILE_SIZE = 15 * 1024 * 1024;

export type ReturnOperation = "eligibility" | "create" | "detail" | "supplement";

export interface ReturnRouteValidation {
  operation?: ReturnOperation;
  orderId?: string;
  caseId?: string;
  mockScenario?: string;
  error?: { field: string; message: string };
}

export function parseReturnRoute(method: string, path: string[], params: URLSearchParams): ReturnRouteValidation {
  const mockScenario = params.get("mockScenario") || undefined;
  if ([...params.keys()].some((key) => key !== "mockScenario")) return { error: { field: "query", message: "Query không được hỗ trợ cho hồ sơ hậu mãi." } };
  if (mockScenario && !SCENARIO_PATTERN.test(mockScenario)) return { error: { field: "mockScenario", message: "Mock scenario không hợp lệ." } };
  if (method === "POST" && path.length === 0) return { operation: "create", mockScenario };
  if (method === "GET" && path.length === 3 && path[0] === "orders" && path[2] === "eligibility" && ID_PATTERN.test(path[1])) return { operation: "eligibility", orderId: path[1], mockScenario };
  if (method === "GET" && path.length === 1 && ID_PATTERN.test(path[0])) return { operation: "detail", caseId: path[0], mockScenario };
  if (method === "POST" && path.length === 2 && path[1] === "supplements" && ID_PATTERN.test(path[0])) return { operation: "supplement", caseId: path[0], mockScenario };
  return { error: { field: "path", message: "Return route is not available." } };
}

function cleanText(value: unknown) { return typeof value === "string" ? value.trim() : ""; }

export function validateCreateReturnCase(value: unknown): { data?: CreateReturnCaseInput; errors: Array<{ field: string; message: string }> } {
  if (!value || typeof value !== "object" || Array.isArray(value)) return { errors: [{ field: "body", message: "Hồ sơ hậu mãi không hợp lệ." }] };
  const source = value as Record<string, unknown>;
  const errors: Array<{ field: string; message: string }> = [];
  const allowed = new Set(["orderId", "lines", "reasonCode", "preferredResolution", "details", "pickupNote", "evidence", "idempotencyKey"]);
  if (Object.keys(source).some((key) => !allowed.has(key))) errors.push({ field: "body", message: "Hồ sơ chứa trường không được hỗ trợ." });
  const orderId = cleanText(source.orderId);
  if (!ID_PATTERN.test(orderId)) errors.push({ field: "orderId", message: "Order không hợp lệ." });

  const seen = new Set<string>();
  const lines = Array.isArray(source.lines) ? source.lines.flatMap((item, index) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) { errors.push({ field: `lines.${index}`, message: "Dòng hàng không hợp lệ." }); return []; }
    const line = item as Record<string, unknown>;
    const lineId = cleanText(line.lineId);
    const quantity = Number(line.quantity);
    if (!ID_PATTERN.test(lineId) || seen.has(lineId) || !Number.isInteger(quantity) || quantity < 1 || quantity > 99) { errors.push({ field: `lines.${index}`, message: "Dòng hàng hoặc số lượng không hợp lệ." }); return []; }
    seen.add(lineId);
    return [{ lineId, quantity }];
  }) : [];
  if (!lines.length || lines.length > 20) errors.push({ field: "lines", message: "Hãy chọn từ 1 đến 20 dòng hàng." });

  const reasonCode = cleanText(source.reasonCode) as ReturnReason;
  const preferredResolution = cleanText(source.preferredResolution) as ReturnResolution;
  const details = cleanText(source.details);
  const pickupNote = cleanText(source.pickupNote);
  if (!REASONS.has(reasonCode)) errors.push({ field: "reasonCode", message: "Hãy chọn lý do hợp lệ." });
  if (!RESOLUTIONS.has(preferredResolution)) errors.push({ field: "preferredResolution", message: "Hãy chọn phương án mong muốn." });
  if (details.length < 10 || details.length > 1000) errors.push({ field: "details", message: "Mô tả cần 10–1000 ký tự." });
  if (pickupNote.length > 300) errors.push({ field: "pickupNote", message: "Ghi chú thu hồi tối đa 300 ký tự." });

  const evidence = Array.isArray(source.evidence) ? source.evidence.flatMap((item, index) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) { errors.push({ field: `evidence.${index}`, message: "Tệp bằng chứng không hợp lệ." }); return []; }
    const file = item as Record<string, unknown>;
    const clientReference = cleanText(file.clientReference);
    const fileName = cleanText(file.fileName);
    const mediaType = cleanText(file.mediaType) as EvidenceMetadata["mediaType"];
    const sizeBytes = Number(file.sizeBytes);
    if (!ID_PATTERN.test(clientReference) || !fileName || fileName.length > 200 || !MEDIA_TYPES.has(mediaType) || !Number.isInteger(sizeBytes) || sizeBytes < 1 || sizeBytes > MAX_FILE_SIZE) { errors.push({ field: `evidence.${index}`, message: "Tệp không đáp ứng chính sách loại hoặc kích thước." }); return []; }
    return [{ clientReference, fileName, mediaType, sizeBytes }];
  }) : [];
  if (evidence.length > 5 || (Array.isArray(source.evidence) && source.evidence.length > 5)) errors.push({ field: "evidence", message: "Tối đa 5 tệp bằng chứng." });
  const idempotencyKey = cleanText(source.idempotencyKey);
  if (!CREATE_KEY_PATTERN.test(idempotencyKey)) errors.push({ field: "idempotencyKey", message: "Khóa chống gửi lặp không hợp lệ." });
  if (errors.length) return { errors };
  return { data: { orderId, lines, reasonCode, preferredResolution, details, pickupNote: pickupNote || undefined, evidence, idempotencyKey }, errors: [] };
}

export function validateSupplement(value: unknown) {
  if (!value || typeof value !== "object" || Array.isArray(value)) return { errors: [{ field: "body", message: "Nội dung bổ sung không hợp lệ." }] };
  const source = value as Record<string, unknown>;
  const message = cleanText(source.message);
  const idempotencyKey = cleanText(source.idempotencyKey);
  const errors: Array<{ field: string; message: string }> = [];
  if (Object.keys(source).some((key) => !["message", "idempotencyKey"].includes(key))) errors.push({ field: "body", message: "Nội dung chứa trường không được hỗ trợ." });
  if (message.length < 2 || message.length > 1000) errors.push({ field: "message", message: "Nội dung cần 2–1000 ký tự." });
  if (!SUPPLEMENT_KEY_PATTERN.test(idempotencyKey)) errors.push({ field: "idempotencyKey", message: "Khóa chống gửi lặp không hợp lệ." });
  return errors.length ? { errors } : { data: { message, idempotencyKey }, errors };
}

export const returnEvidencePolicy = { maxFiles: 5, maxFileSize: MAX_FILE_SIZE, mediaTypes: MEDIA_TYPES };
