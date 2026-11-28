import { traceCode } from "@/lib/customer-core/trace-code";

export interface OrderBatchTrace {
  order_line_id: string; sku_code: string; batch_id: string; batch_code: string; packed_quantity: number;
  status: "AVAILABLE" | "NOT_AVAILABLE" | "NO_LONGER_PUBLIC" | "UNAVAILABLE"; qr_url: string | null;
}
export interface OrderTraceability {
  order_id: string; state: "READY" | "PACKING_PENDING" | "NOT_APPLICABLE" | "UNAVAILABLE";
  items: OrderBatchTrace[]; total: number;
}
const UUID = /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/u;
const record = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === "object" && !Array.isArray(value);
export function isOrderTraceability(value: unknown, id: string, origins: string[]): value is OrderTraceability {
  if (!record(value) || value.order_id !== id || !Array.isArray(value.items) || value.items.length > 1000 || value.total !== value.items.length) return false;
  if (value.state !== "READY") return ["PACKING_PENDING", "NOT_APPLICABLE", "UNAVAILABLE"].includes(String(value.state)) && value.items.length === 0;
  if (!value.items.length) return false;
  const rows = new Set<string>(); const lines = new Map<string, string>(); const skus = new Map<string, string>();
  return value.items.every((row: unknown) => {
    if (!record(row) || typeof row.order_line_id !== "string" || !UUID.test(row.order_line_id) || typeof row.batch_id !== "string" || !UUID.test(row.batch_id) || typeof row.batch_code !== "string" || !row.batch_code.trim() || row.batch_code.length > 100 || typeof row.sku_code !== "string" || !row.sku_code || row.sku_code.length > 100 || !Number.isInteger(row.packed_quantity) || Number(row.packed_quantity) < 1 || Number(row.packed_quantity) > 2147483647) return false;
    if (rows.has(`${row.order_line_id}:${row.batch_id}`) || lines.has(row.order_line_id) && lines.get(row.order_line_id) !== row.sku_code || skus.has(row.sku_code) && skus.get(row.sku_code) !== row.order_line_id) return false;
    rows.add(`${row.order_line_id}:${row.batch_id}`); lines.set(row.order_line_id, row.sku_code); skus.set(row.sku_code, row.order_line_id);
    if (row.status === "AVAILABLE" || row.status === "NO_LONGER_PUBLIC") return typeof row.qr_url === "string" && row.qr_url.startsWith("https://") && Boolean(traceCode(row.qr_url, origins));
    return (row.status === "NOT_AVAILABLE" || row.status === "UNAVAILABLE") && row.qr_url === null;
  });
}

export function matchesFrozenOrder(value: OrderTraceability, lines: Array<{ skuCode?: string; quantity: number }>): boolean {
  if (value.state !== "READY") return true;
  const quantities = new Map<string, number>();
  for (const row of value.items) quantities.set(row.sku_code, (quantities.get(row.sku_code) ?? 0) + row.packed_quantity);
  return lines.length === quantities.size && new Set(lines.map((line) => line.skuCode)).size === lines.length && lines.every((line) => line.skuCode && quantities.get(line.skuCode) === line.quantity);
}
