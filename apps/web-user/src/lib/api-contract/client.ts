import { PublicApiError, normalizePublicError } from "./error";
import { setPublicIdempotencyKey } from "./idempotency";

export interface PublicJsonRequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  idempotencyKey?: string;
  ifMatch?: string;
}

export interface PublicApiResponse<T> {
  data: T;
  status: number;
  headers: Headers;
  etag: string | null;
  retryAfterSeconds?: number;
}

function retryAfterSeconds(value: string | null): number | undefined {
  if (!value) return undefined;
  if (/^[0-9]+$/u.test(value)) return Number(value);
  const at = Date.parse(value);
  if (Number.isNaN(at)) return undefined;
  return Math.max(0, Math.ceil((at - Date.now()) / 1000));
}

async function readPayload(response: Response): Promise<unknown> {
  if (response.status === 204) return undefined;
  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.includes("application/json")) return undefined;
  try {
    return await response.json() as unknown;
  } catch {
    return undefined;
  }
}

export async function requestPublicJson<T>(input: RequestInfo | URL, options: PublicJsonRequestOptions = {}): Promise<PublicApiResponse<T>> {
  const { body, idempotencyKey, ifMatch, ...init } = options;
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (body !== undefined) headers.set("Content-Type", "application/json");
  if (idempotencyKey) setPublicIdempotencyKey(headers, idempotencyKey);
  if (ifMatch) headers.set("If-Match", ifMatch);

  const response = await fetch(input, {
    ...init,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    cache: init.cache ?? "no-store"
  });
  const payload = await readPayload(response);

  if (!response.ok) {
    throw new PublicApiError(response.status, normalizePublicError(payload, {
      requestId: response.headers.get("x-request-id") ?? undefined
    }));
  }

  const retryAfter = retryAfterSeconds(response.headers.get("retry-after"));
  return {
    data: payload as T,
    status: response.status,
    headers: response.headers,
    etag: response.headers.get("etag"),
    ...(retryAfter !== undefined ? { retryAfterSeconds: retryAfter } : {})
  };
}
