import type { PublicErrorBody } from "./types";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function text(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

export function normalizePublicError(
  value: unknown,
  fallback: { code?: string; message?: string; requestId?: string } = {}
): PublicErrorBody {
  const body = isRecord(value) ? value : {};
  return {
    code: text(body.code) ?? text(body.error_code) ?? fallback.code ?? "PUBLIC_API_ERROR",
    message: text(body.message) ?? text(body.user_message) ?? fallback.message ?? "Không thể xử lý yêu cầu lúc này.",
    requestId: text(body.requestId) ?? text(body.request_id) ?? fallback.requestId ?? "unknown",
    ...(typeof body.retryable === "boolean" ? { retryable: body.retryable } : {}),
    ...(text(body.operation_id) ? { operation_id: text(body.operation_id) } : {}),
    ...(isRecord(body.details)
      ? { details: body.details }
      : Array.isArray(body.errors)
        ? { details: { errors: body.errors } }
        : {})
  };
}

export class PublicApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId: string;
  readonly retryable: boolean;
  readonly operationId?: string;
  readonly details?: Record<string, unknown>;

  constructor(status: number, body: PublicErrorBody) {
    super(body.message);
    this.name = "PublicApiError";
    this.status = status;
    this.code = body.code;
    this.requestId = body.requestId;
    this.retryable = body.retryable ?? false;
    this.operationId = body.operation_id;
    this.details = body.details;
  }

  get isUnauthorized(): boolean {
    return this.status === 401;
  }

  get isForbidden(): boolean {
    return this.status === 403;
  }
}

export class ContractDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ContractDataError";
  }
}
