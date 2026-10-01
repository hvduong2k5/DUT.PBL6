import { parseCustomerCapabilityProjection, type CustomerCapabilityProjection } from "./capabilities";

export class CapabilityProjectionError extends Error {
  constructor(readonly status: number, message: string) {
    super(message);
    this.name = "CapabilityProjectionError";
  }
}

export async function fetchCustomerCapabilities(accessToken?: string): Promise<CustomerCapabilityProjection> {
  const base = (process.env.CUSTOMER_EXTENSIONS_UPSTREAM_URL ?? "http://127.0.0.1:4020/api/v1").replace(/\/$/u, "");
  const headers = new Headers({ Accept: "application/json", "X-Customer-Actor": accessToken ? "REGISTERED" : "GUEST" });
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  const response = await fetch(`${base}/customer/capabilities`, { headers, cache: "no-store", redirect: "manual" });
  if (!response.ok) throw new CapabilityProjectionError(response.status, "Không thể tải quyền Customer.");
  const projection = parseCustomerCapabilityProjection(await response.json());
  if (!projection) throw new CapabilityProjectionError(502, "Dữ liệu quyền Customer không hợp lệ.");
  return projection;
}
