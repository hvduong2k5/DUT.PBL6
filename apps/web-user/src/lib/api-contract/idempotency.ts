import { ContractDataError } from "./error";

export const PUBLIC_IDEMPOTENCY_HEADER = "Idempotency-Key";
export const ORDER_PROVIDER_IDEMPOTENCY_HEADER = "X-Idempotency-Key";

const IDEMPOTENCY_KEY = /^[!-~]{16,128}$/u;
const KEY_PREFIX = /^[A-Za-z0-9._~-]{2,40}$/u;

export function isValidIdempotencyKey(value: string): boolean {
  return IDEMPOTENCY_KEY.test(value);
}

export function assertIdempotencyKey(value: string): string {
  if (!isValidIdempotencyKey(value)) {
    throw new ContractDataError("Idempotency-Key must contain 16-128 visible ASCII characters.");
  }
  return value;
}

export function createIdempotencyKey(prefix = "web"): string {
  if (!KEY_PREFIX.test(prefix)) throw new ContractDataError("Idempotency key prefix is invalid.");
  if (!globalThis.crypto?.randomUUID) throw new ContractDataError("Secure UUID generation is unavailable.");
  return assertIdempotencyKey(`${prefix}-${globalThis.crypto.randomUUID()}`);
}

export function readPublicIdempotencyKey(headers: Headers): string | undefined {
  const value = headers.get(PUBLIC_IDEMPOTENCY_HEADER);
  return value ? assertIdempotencyKey(value) : undefined;
}

export function setPublicIdempotencyKey(headers: Headers, value: string): void {
  headers.set(PUBLIC_IDEMPOTENCY_HEADER, assertIdempotencyKey(value));
}

export function setOrderProviderIdempotencyKey(headers: Headers, value: string): void {
  headers.set(ORDER_PROVIDER_IDEMPOTENCY_HEADER, assertIdempotencyKey(value));
}
