import { parseCustomerCapabilityProjection, type CustomerCapabilityProjection } from "@/lib/auth/capabilities";

export async function getCustomerCapabilities(): Promise<CustomerCapabilityProjection> {
  const response = await fetch("/api/capabilities", { credentials: "include", cache: "no-store" });
  const payload: unknown = await response.json();
  if (!response.ok) throw new Error(payload && typeof payload === "object" && "message" in payload && typeof payload.message === "string" ? payload.message : "Không thể kiểm tra quyền Customer.");
  const projection = parseCustomerCapabilityProjection(payload);
  if (!projection) throw new Error("Dữ liệu quyền Customer không hợp lệ.");
  return projection;
}
