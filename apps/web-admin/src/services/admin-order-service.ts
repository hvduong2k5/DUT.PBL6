import type { AdminOrderDetail, AdminOrderList } from "@/lib/orders/types";
import { AdminOrderApiError } from "@/lib/orders/types";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`/api/admin/orders${path}`, { cache: "no-store", credentials: "include" });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminOrderApiError(response.status, payload);
  return payload as T;
}

export const adminOrderService = {
  list() { return request<AdminOrderList>(""); },
  detail(orderId: string) { return request<AdminOrderDetail>(`/${encodeURIComponent(orderId)}`); }
};
