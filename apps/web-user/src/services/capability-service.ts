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
