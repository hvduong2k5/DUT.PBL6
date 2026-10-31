import type { ShipmentDetail, ShipmentList } from "@/lib/shipping/types";
import { AdminShippingApiError } from "@/lib/shipping/types";

async function request<T>(path: string): Promise<T> {
  const response = await fetch(`/api/admin/shipping${path}`, { cache: "no-store", credentials: "include" });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminShippingApiError(response.status, payload);
  return payload as T;
}

export const adminShippingService = {
  shipments() { return request<ShipmentList>("/shipments"); },
  shipment(shipmentId: string) { return request<ShipmentDetail>(`/shipments/${encodeURIComponent(shipmentId)}`); },
  assignedShipments() { return request<ShipmentList>("/workbench/shipments"); },
  assignedShipment(shipmentId: string) { return request<ShipmentDetail>(`/workbench/shipments/${encodeURIComponent(shipmentId)}`); }
};
