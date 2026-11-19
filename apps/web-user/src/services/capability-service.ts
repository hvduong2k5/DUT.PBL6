import { parseCustomerCapabilityProjection, type CustomerCapabilityProjection } from "@/lib/auth/capabilities";

export class CapabilityApiError extends Error {
  constructor(readonly status: number, readonly code: string, message: string) {
    super(message);
    this.name = "CapabilityApiError";
  }
}

export async function getCustomerCapabilities(): Promise<CustomerCapabilityProjection> {
  const response = await fetch("/api/capabilities", { credentials: "include", cache: "no-store" });
  const payload: unknown = await response.json();
  if (!response.ok) {
    const source = payload && typeof payload === "object" ? payload as Record<string, unknown> : {};
    throw new CapabilityApiError(
      response.status,
      typeof source.code === "string" ? source.code : "CAPABILITY_UNAVAILABLE",
      typeof source.message === "string" ? source.message : "Không thể kiểm tra quyền Customer."
    );
  }
  const projection = parseCustomerCapabilityProjection(payload);
  if (!projection) throw new CapabilityApiError(502, "CAPABILITY_INVALID", "Dữ liệu quyền Customer không hợp lệ.");
  return projection;
}

export type CapabilityFailureAction = "forbidden" | "session" | "error";

const SESSION_FAILURE_CODES = new Set(["AUTH_REQUIRED", "SESSION_EXPIRED", "SESSION_INVALID"]);

/**
 * How the session provider should react to a failed capability check.
 * A bare 401 is not proof the session ended: only the explicit session
 * codes count as a session failure. Any other 401 is a generic error, so a
 * proxied upstream failure can never masquerade as "forbidden" — or
 * silently log the user out.
 */
export function capabilityFailureAction(status: number, code: string): CapabilityFailureAction {
  if (status === 403) return "forbidden";
  if (status === 401 && SESSION_FAILURE_CODES.has(code)) return "session";
  return "error";
}
