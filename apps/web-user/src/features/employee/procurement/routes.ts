/** Only the approved Procurement surface is reachable. No supply-plan fallback. */
const uuid = "(?!00000000-0000-0000-0000-000000000000)[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}";
const rules: Array<[string, string]> = [
  ["GET|POST", "suppliers|materials|purchase-orders"],
  ["GET|PUT", `(?:suppliers|materials|purchase-orders)/${uuid}`],
  ["PATCH", `(?:suppliers|materials)/${uuid}/status`],
  ["GET", `(?:suppliers|materials)/${uuid}/offers`],
  ["PUT", `materials/${uuid}/preferred-supplier`],
  ["POST", `suppliers/${uuid}/offers`],
  ["PATCH", `suppliers/${uuid}/offers/${uuid}/status`],
  ["GET|POST", `suppliers/${uuid}/contracts`],
  ["GET", `contracts/${uuid}`],
  ["POST", `contracts/${uuid}/(?:activate|terminate)`],
  ["GET|POST", `purchase-orders/${uuid}/revisions`],
  ["POST", `purchase-orders/${uuid}/(?:submit|approve|reject|send|supplier-response|cancel|close)`],
  ["POST", "goods-receipts"],
  ["GET", `goods-receipts/${uuid}`],
  ["POST", `goods-receipts/${uuid}/inspect`],
  ["POST", `goods-receipts/${uuid}/variances/${uuid}/resolve`],
];

export function procurementRoute(method: string, path: string): boolean {
  return rules.some(([methods, pattern]) => methods.split("|").includes(method) && new RegExp(`^(?:${pattern})$`, "u").test(path));
}

export function procurementQuery(path: string, params: URLSearchParams): string | undefined {
  const paged = ["suppliers", "materials", "purchase-orders"].includes(path) || /\/(?:contracts|revisions)$/u.test(path);
  const allowed = new Set(paged ? ["limit", "offset", ...(path === "purchase-orders" ? ["supplier_id", "status"] : [])] : []);
  const query = new URLSearchParams();
  for (const [name, value] of params) {
    if (!allowed.has(name) || query.has(name) || value.length > 128) return undefined;
    if (name === "limit" && (!/^[1-9][0-9]*$/u.test(value) || Number(value) > 100)) return undefined;
    if (name === "offset" && (!/^(?:0|[1-9][0-9]*)$/u.test(value) || !Number.isSafeInteger(Number(value)))) return undefined;
    query.set(name, value);
  }
  return query.toString();
}
