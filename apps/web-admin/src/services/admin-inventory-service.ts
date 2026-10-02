import type { InventoryAdjustmentInput, InventoryIssueInput, InventoryOperationResult, InventoryReceiptInput, InventorySkuDetail, InventorySkuList, WarehouseWorkbenchContext } from "@/lib/inventory/types";
import { AdminInventoryApiError } from "@/lib/inventory/types";

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`/api/admin/inventory${path}`, { ...init, cache: "no-store", credentials: "include" });
  const payload = (response.headers.get("content-type") ?? "").includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new AdminInventoryApiError(response.status, payload);
  return payload as T;
}

export const adminInventoryService = {
  skus() { return request<InventorySkuList>("/skus"); },
  sku(skuId: string) { return request<InventorySkuDetail>(`/skus/${encodeURIComponent(skuId)}`); },
  workbench() { return request<WarehouseWorkbenchContext>("/workbench"); },
  receive(input: InventoryReceiptInput) { return request<InventoryOperationResult>("/workbench/receipts", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }); },
  issue(input: InventoryIssueInput) { return request<InventoryOperationResult>("/workbench/issues", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }); },
  adjust(input: InventoryAdjustmentInput) { return request<InventoryOperationResult>("/workbench/adjustments", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }); }
};
