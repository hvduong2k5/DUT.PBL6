import fs from "node:fs";
import path from "node:path";

const root = process.cwd();
const outputDir = path.join(root, "docs", "03_api_specs", "frontend-contracts");
fs.mkdirSync(outputDir, { recursive: true });

const readJson = (relativePath) => JSON.parse(fs.readFileSync(path.join(root, relativePath), "utf8"));
const customerSources = [
  readJson("mocks/mockoon/mobile_pbl.json"),
  readJson("mocks/mockoon/customer_extensions.json")
];
const adminSources = [readJson("mocks/mockoon/admin_extensions.json")];

const quote = (value) => JSON.stringify(value);
const scalar = (value) => {
  if (value === null) return "null";
  if (typeof value === "string") return quote(value);
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return quote(String(value));
};

function yaml(value, indent = 0) {
  const pad = " ".repeat(indent);
  if (value === null || typeof value !== "object") return scalar(value);
  if (Array.isArray(value)) {
    if (value.length === 0) return "[]";
    return value.map((item) => {
      if (item !== null && typeof item === "object") {
        if (!Array.isArray(item) && Object.keys(item).length === 0) return `${pad}- {}`;
        return `${pad}-\n${yaml(item, indent + 2)}`;
      }
      return `${pad}- ${scalar(item)}`;
    }).join("\n");
  }
  const entries = Object.entries(value);
  if (entries.length === 0) return "{}";
  return entries.map(([key, item]) => {
    const renderedKey = /^[A-Za-z_$.-][A-Za-z0-9_$.-]*$/.test(key) ? key : quote(key);
    if (item !== null && typeof item === "object") {
      const rendered = yaml(item, indent + 2);
      if (rendered === "[]" || rendered === "{}") return `${pad}${renderedKey}: ${rendered}`;
      return `${pad}${renderedKey}:\n${rendered}`;
    }
    return `${pad}${renderedKey}: ${scalar(item)}`;
  }).join("\n");
}

function inferFormat(value) {
  if (typeof value !== "string") return undefined;
  if (/^https?:\/\//i.test(value)) return "uri";
  if (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) return "email";
  if (/^\d{4}-\d{2}-\d{2}T/.test(value)) return "date-time";
  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) return "date";
  return undefined;
}

function inferSchema(value, key = "") {
  if (value === null) return { type: ["string", "null"], description: `${key || "Value"} may be null.` };
  if (Array.isArray(value)) {
    const samples = value.map((item) => inferSchema(item, key));
    return { type: "array", items: samples.length === 0 ? {} : samples.reduce(mergeSchemas) };
  }
  if (typeof value === "object") {
    const properties = {};
    for (const [childKey, child] of Object.entries(value)) properties[childKey] = inferSchema(child, childKey);
    return { type: "object", required: Object.keys(value), properties, additionalProperties: false };
  }
  if (typeof value === "boolean") return { type: "boolean" };
  if (typeof value === "number") return { type: Number.isInteger(value) ? "integer" : "number" };
  const schema = { type: "string" };
  const format = inferFormat(value);
  if (format) schema.format = format;
  return schema;
}

function schemaTypes(schema) {
  if (!schema?.type) return [];
  return Array.isArray(schema.type) ? schema.type : [schema.type];
}

function mergeSchemas(left, right) {
  if (!left || Object.keys(left).length === 0) return right ?? {};
  if (!right || Object.keys(right).length === 0) return left;
  const types = [...new Set([...schemaTypes(left), ...schemaTypes(right)])];
  if (types.includes("object") && types.every((type) => type === "object" || type === "null")) {
    const properties = {};
    for (const key of new Set([...Object.keys(left.properties ?? {}), ...Object.keys(right.properties ?? {})])) {
      const a = left.properties?.[key];
      const b = right.properties?.[key];
      properties[key] = a && b ? mergeSchemas(a, b) : (a ?? b);
    }
    const rightRequired = new Set(right.required ?? []);
    const required = (left.required ?? []).filter((key) => rightRequired.has(key));
    return { type: types.length === 1 ? "object" : types, ...(required.length ? { required } : {}), properties, additionalProperties: false };
  }
  if (types.includes("array") && types.every((type) => type === "array" || type === "null")) {
    return { type: types.length === 1 ? "array" : types, items: mergeSchemas(left.items ?? {}, right.items ?? {}) };
  }
  const merged = { type: types.length === 1 ? types[0] : types };
  if (left.format && left.format === right.format) merged.format = left.format;
  return merged;
}

function parseBody(body) {
  if (!body || !String(body).trim()) return {};
  try { return JSON.parse(body); } catch { return { raw: String(body) }; }
}

function normalizeStablePayload(apiPath, payload) {
  const cloned = structuredClone(payload);
  const removeLegacyDistrict = (value) => {
    if (Array.isArray(value)) return value.map(removeLegacyDistrict);
    if (!value || typeof value !== "object") return value;
    const result = {};
    for (const [key, child] of Object.entries(value)) {
      if (key !== "district") result[key] = removeLegacyDistrict(child);
    }
    return result;
  };
  const normalized = removeLegacyDistrict(cloned);
  if (apiPath.startsWith("/api/v1/cart") && Array.isArray(normalized.items)) {
    let totalItems = 0;
    let subtotal = 0;
    for (const item of normalized.items) {
      const quantity = Number(item.quantity ?? 0);
      const unit = Number(item.unit_price?.units ?? 0);
      totalItems += quantity;
      subtotal += unit * quantity;
      if (item.subtotal && typeof item.subtotal === "object") item.subtotal.units = unit * quantity;
    }
    normalized.total_items = totalItems;
    if (normalized.subtotal_amount && typeof normalized.subtotal_amount === "object") normalized.subtotal_amount.units = subtotal;
  }
  return normalized;
}

function successResponse(route) {
  const successes = route.responses.filter((response) => response.statusCode >= 200 && response.statusCode < 300);
  return successes.find((response) => response.default) ?? successes.at(-1) ?? route.responses.at(-1);
}

function normalizePath(endpoint) {
  const raw = endpoint.startsWith("api/v1/") ? endpoint : `api/v1/${endpoint}`;
  return `/${raw}`.replace(/:([A-Za-z0-9_]+)/g, "{$1}");
}

function operationId(method, apiPath) {
  const words = apiPath.replace(/^\/api\/v1\//, "").replace(/[{}]/g, "").split(/[^A-Za-z0-9]+/).filter(Boolean);
  return `${method.toLowerCase()}${words.map((word) => word[0].toUpperCase() + word.slice(1)).join("")}`;
}

function tagFor(apiPath, audience) {
  const part = apiPath.replace(/^\/api\/v1\//, "").split("/")[audience === "admin" ? 1 : 0] ?? "General";
  const labels = {
    auth: "Authentication", categories: "Catalog", products: "Catalog", cart: "Cart", checkout: "Checkout",
    orders: "Orders", profile: "Customer profile", promotions: "Promotions", loyalty: "Loyalty",
    reviews: "Reviews", returns: "After-sales", trace: "Traceability", payments: "Payments",
    webhooks: "Partner webhooks", support: "Customer support", customer: "Customer capabilities",
    locations: "Locations", b2b: "B2B", session: "Session", access: "Access management",
    catalog: "Product management", inventory: "Inventory", packing: "Packing", shipping: "Shipping"
  };
  return labels[part] ?? part[0].toUpperCase() + part.slice(1);
}

function backlogFor(apiPath, audience) {
  const key = audience === "admin" ? apiPath.replace(/^\/api\/v1\/admin\//, "").split("/")[0] : apiPath.replace(/^\/api\/v1\//, "").split("/")[0];
  const customer = {
    auth: ["EPIC_01"], categories: ["EPIC_03"], products: ["EPIC_03", "EPIC_04"], cart: ["EPIC_05"],
    checkout: ["EPIC_06"], orders: ["EPIC_08"], profile: ["EPIC_02"], promotions: ["EPIC_17"],
    loyalty: ["EPIC_17"], reviews: ["EPIC_15"], returns: ["EPIC_14"], trace: ["EPIC_19"],
    payments: ["EPIC_07"], webhooks: ["EPIC_11", "EPIC_12"], support: ["EPIC_16"],
    customer: ["EPIC_01", "EPIC_18"], locations: ["EPIC_02", "EPIC_06"], b2b: ["EPIC_18"],
    media: ["EPIC_14", "EPIC_16", "EPIC_18"], notifications: ["EPIC_28"], "notification-preferences": ["EPIC_28"]
  };
  const admin = {
    session: ["EPIC_22", "EPIC_23"], access: ["EPIC_22"], b2b: ["EPIC_18"], catalog: ["EPIC_04"],
    inventory: ["EPIC_09"], orders: ["EPIC_08"], packing: ["EPIC_10"], shipping: ["EPIC_11"],
    payments: ["EPIC_07", "EPIC_24"], returns: ["EPIC_14"], support: ["EPIC_16"]
  };
  return (audience === "admin" ? admin : customer)[key] ?? ["BACKLOG_REVIEW_REQUIRED"];
}

const requestExamples = {
  "POST /api/v1/auth/register": { phone_number: "0905123456", full_name: "Nguyen Van An", email: "an@example.com", password: "StrongPass!123" },
  "POST /api/v1/auth/login": { identifier: "0905123456", password: "StrongPass!123" },
  "POST /api/v1/auth/refresh": { refresh_token: "opaque-refresh-token" },
  "POST /api/v1/auth/recovery-requests": { email: "an@example.com" },
  "POST /api/v1/auth/recovery-requests/verify": { recoveryToken: "recovery-token" },
  "POST /api/v1/auth/recovery-requests/reset": { resetProof: "reset-proof", newPassword: "NewStrongPass!123" },
  "POST /api/v1/cart/items": { sku_code: "MX-GION-500G", quantity: 2 },
  "PUT /api/v1/cart/items/{item_id}": { quantity: 3 },
  "POST /api/v1/checkout": { channel: "D2C_WEB", shipping_address: { recipient_name: "Nguyen Van An", phone_number: "0905123456", street_address: "128 Le Loi", ward: "Phuong Thuan Hoa", province: "Thanh pho Hue" }, payment_method: "VIETQR", items: [{ sku_code: "MX-GION-500G", quantity: 2 }] },
  "POST /api/v1/orders/{order_id}/cancel": { reason: "CUSTOMER_CHANGED_MIND" },
  "PUT /api/v1/profile": { full_name: "Nguyen Van An", email: "an@example.com" },
  "POST /api/v1/profile/addresses": { recipient_name: "Nguyen Van An", phone_number: "0905123456", street_address: "128 Le Loi", ward: "Phuong Thuan Hoa", province: "Thanh pho Hue", is_default: true },
  "PUT /api/v1/profile/addresses/{address_id}": { recipient_name: "Nguyen Van An", phone_number: "0905123456", street_address: "130 Le Loi", ward: "Phuong Thuan Hoa", province: "Thanh pho Hue", is_default: true },
  "POST /api/v1/promotions/validate": { voucher_code: "OMAMA_FREESHIP_30K", subtotal_amount: 250000 },
  "POST /api/v1/reviews": { order_id: "ORD-20261015-0042", sku_code: "MX-GION-500G", rating: 5, comment: "San pham tot", media_urls: [] },
  "POST /api/v1/returns": { order_id: "ORD-20261015-0042", lines: [{ line_id: "LINE-01", quantity: 1 }], reason_code: "DAMAGED_IN_TRANSIT", details: "Bao bi bi rach", evidence_media_urls: ["https://cdn.example.com/evidence.jpg"], refund_bank_code: "MBBANK", refund_account_number: "0905123456", refund_account_holder: "NGUYEN VAN AN", idempotency_key: "6f5902ac-2377-4d4f-8b32-17cb34f29450" },
  "POST /api/v1/orders/guest-access/challenges": { orderNumber: "ORD-20261015-0042", contact: "0905123456" },
  "POST /api/v1/orders/guest-access/challenges/{challengeId}/verify": { otp: "123456" },
  "POST /api/v1/orders/guest-access/grants": { orderId: "ORD-20261015-0042", contact: "0905123456" },
  "POST /api/v1/support/tickets": { subjectCode: "ORDER_HELP", title: "Can ho tro don hang", message: "Vui long kiem tra don hang", requestedPriority: "NORMAL", linkedOrderId: "ORD-20261015-0042", attachments: [], idempotencyKey: "6f5902ac-2377-4d4f-8b32-17cb34f29450" },
  "POST /api/v1/support/tickets/{ticketId}/messages": { message: "Toi da bo sung thong tin", attachments: [], idempotencyKey: "294dd630-8362-4dcb-a0fd-84dd3fd52f67" },
  "POST /api/v1/b2b/quote-requests": { organizationId: "ORG-FPT-MT", purpose: "VIP_CUSTOMER_GIFT", requestedDeliveryDate: "2026-11-15", deliveryProvinceCode: "48", deliveryProvinceName: "Thanh pho Da Nang", items: [{ skuId: "SKU-GIFT-ROYAL", quantity: 80 }], branding: { engraveLogo: true, customSleeve: true, greetingCard: true, ribbon: false, attachments: [] }, invoiceRequested: true, sampleRequested: false, idempotencyKey: "60d69078-d6c4-4b4e-a042-f2452dc155c5" },
  "POST /api/v1/b2b/quotes/{quoteId}/accept": { confirmation: true, idempotencyKey: "482b2f90-6f68-47f3-9bf2-1022ee5b173a" },
  "POST /api/v1/payments/vietqr/callback": { transaction_id: "FT2628891048201", order_id: "ORD-20261015-0042", amount: 345000, bank_code: "MBBANK", account_number: "0123456789", paid_at: "2026-10-15T08:30:12Z" },
  "POST /api/v1/webhooks/marketplace/shopee": { event_id: "SHP-EVT-01", platform: "SHOPEE", order_sn: "2610159AKJ3812", status: "READY_TO_SHIP", buyer_username: "customer_hue", order_items: [{ ext_item_id: "192837465", ext_model_id: "99881122", quantity: 2, price: 110000 }] },
  "POST /api/v1/webhooks/marketplace/tiktok": { event_id: "TT-EVT-01", platform: "TIKTOK", order_sn: "TT-261015-001", status: "READY_TO_SHIP", buyer_username: "customer_hue", order_items: [{ ext_item_id: "192837465", ext_model_id: "99881122", quantity: 2, price: 110000 }] },
  "POST /api/v1/webhooks/logistics/ghn": { event_id: "GHN-EVT-01", OrderCode: "GHN202610150998", Status: "delivered", Time: "2026-10-16T15:30:00Z", CurrentWarehouse: "Buu cuc Hue", Note: "Delivered" },
  "POST /api/v1/admin/b2b/quote-requests/{requestId}/actions": { action: "ASSIGN", expectedRevision: 4, assigneeId: "EMP-0007", idempotencyKey: "2c20501b-56a5-4f18-adb4-84d42eca6e03" },
  "PATCH /api/v1/admin/packing/workbench/tasks/{taskId}/checklist/{itemId}": { checked: true, expectedRevision: 2 },
  "POST /api/v1/admin/packing/workbench/tasks/{taskId}/complete": { expectedRevision: 3, evidenceFileIds: ["FILE-PACK-01"], idempotencyKey: "c5039ef6-acf7-4fd1-9889-51b9084ee7ea" },
  "POST /api/v1/admin/inventory/workbench/receipts": { skuId: "SKU-MX-GION-500G", batchCode: "LOT-20261005-01", quantity: 100, reasonCode: "PRODUCTION_RECEIPT", expectedRevision: 8, idempotencyKey: "45e011ce-c117-4644-99f9-b4f5a9873581" },
  "POST /api/v1/admin/inventory/workbench/issues": { skuId: "SKU-MX-GION-500G", batchCode: "LOT-20261005-01", quantity: 5, reasonCode: "DAMAGE", expectedRevision: 8, idempotencyKey: "52658e3d-38ad-42fa-8f3b-1d64c5a78d9a" },
  "POST /api/v1/admin/inventory/workbench/adjustments": { skuId: "SKU-MX-GION-500G", countedQuantity: 94, reasonCode: "CYCLE_COUNT", note: "Kiem ke cuoi ca", expectedRevision: 8, idempotencyKey: "744eae85-8145-47a0-ad84-314d0e44f9dc" }
};

const publicCustomerPrefixes = [
  "/api/v1/categories", "/api/v1/products", "/api/v1/trace/", "/api/v1/locations/checkout",
  "/api/v1/reviews/products/", "/api/v1/promotions/vouchers", "/api/v1/promotions/validate",
  "/api/v1/auth/register", "/api/v1/auth/login", "/api/v1/auth/recovery-requests", "/api/v1/auth/verification-requests",
  "/api/v1/orders/guest-access/challenges", "/api/v1/webhooks/", "/api/v1/payments/vietqr/callback"
];

function isPublicCustomer(apiPath) {
  return publicCustomerPrefixes.some((prefix) => apiPath === prefix || apiPath.startsWith(prefix));
}

function isListPayload(payload) {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return false;
  return Object.values(payload).some(Array.isArray);
}

function isListOperation(method, apiPath, payload) {
  if (method !== "GET" || !isListPayload(payload)) return false;
  if (/\{[^}]+\}$/.test(apiPath) && !apiPath.includes("/reviews/products/")) return false;
  const singletonPaths = new Set([
    "/api/v1/auth/me", "/api/v1/profile", "/api/v1/customer/capabilities",
    "/api/v1/support/context", "/api/v1/b2b/context", "/api/v1/locations/checkout",
    "/api/v1/loyalty/points", "/api/v1/admin/session", "/api/v1/admin/inventory/workbench"
  ]);
  return !singletonPaths.has(apiPath);
}

function emptyExample(payload) {
  const cloned = structuredClone(payload);
  for (const [key, value] of Object.entries(cloned)) {
    if (Array.isArray(value)) cloned[key] = [];
    if (["total", "total_items", "total_pages", "total_reviews", "filteredCount", "totalCount"].includes(key) && typeof value === "number") cloned[key] = 0;
  }
  return cloned;
}

function responseRef(status) {
  const map = { 400: "BadRequest", 401: "Unauthorized", 403: "Forbidden", 404: "NotFound", 409: "Conflict", 410: "Gone", 422: "UnprocessableEntity", 429: "TooManyRequests", 500: "InternalError", 502: "BadGateway", 503: "ServiceUnavailable" };
  return { $ref: `#/components/responses/${map[status]}` };
}

function statusPolicy(route, apiPath, audience) {
  const method = route.method.toUpperCase();
  const success = successResponse(route)?.statusCode ?? (method === "POST" ? 201 : 200);
  const statuses = new Set([success]);
  const list = method === "GET" && !/\{[^}]+\}/.test(apiPath);
  const detail = /\{[^}]+\}/.test(apiPath);
  if (audience === "admin") {
    statuses.add(401); statuses.add(403);
    if (list) statuses.add(400);
    if (detail) statuses.add(404);
    if (method !== "GET") { statuses.add(400); statuses.add(409); statuses.add(422); }
    statuses.add(500); statuses.add(503);
  } else {
    const isPublic = isPublicCustomer(apiPath);
    if (!isPublic || apiPath.includes("/auth/login")) statuses.add(401);
    if (!isPublic || apiPath.includes("/auth/login")) statuses.add(403);
    if (apiPath.startsWith("/api/v1/webhooks/") || apiPath === "/api/v1/payments/vietqr/callback") statuses.add(401);
    if (list || method !== "GET") statuses.add(400);
    if (detail) statuses.add(404);
    if (method !== "GET") { statuses.add(409); statuses.add(422); }
    if (apiPath.includes("recovery") || apiPath.includes("challenges") || apiPath.includes("messages")) statuses.add(429);
    if (apiPath.includes("verify") || apiPath.includes("reset")) statuses.add(410);
    statuses.add(500); statuses.add(503);
  }
  return [...statuses].sort((a, b) => a - b);
}

function securityFor(apiPath, audience) {
  if (audience === "admin") return [{ bearerAuth: [] }, { adminSession: [] }];
  if (apiPath.startsWith("/api/v1/webhooks/") || apiPath === "/api/v1/payments/vietqr/callback") return [{ webhookSignature: [] }];
  if (apiPath.startsWith("/api/v1/cart") || apiPath === "/api/v1/checkout") return [{ bearerAuth: [] }, { customerSession: [] }, { guestSession: [] }];
  if (/^\/api\/v1\/orders\/\{order_id\}/.test(apiPath)) return [{ bearerAuth: [] }, { customerSession: [] }, { guestGrant: [] }];
  if (apiPath.startsWith("/api/v1/support/") || apiPath === "/api/v1/customer/capabilities") return [{}, { bearerAuth: [] }, { customerSession: [] }, { guestSession: [] }];
  if (isPublicCustomer(apiPath)) return [];
  return [{ bearerAuth: [] }, { customerSession: [] }];
}

function pathParameters(apiPath) {
  return [...apiPath.matchAll(/\{([^}]+)\}/g)].map((match) => ({ name: match[1], in: "path", required: true, schema: { type: "string", minLength: 1 } }));
}

function listQueryParameters() {
  return [
    { name: "q", in: "query", required: false, description: "Case-insensitive search text for supported identifiers and labels.", schema: { type: "string", maxLength: 200 } },
    { name: "status", in: "query", required: false, description: "Domain status filter. Allowed values are defined by the endpoint response status vocabulary.", schema: { type: "string" } },
    { name: "page", in: "query", required: false, schema: { type: "integer", minimum: 1, default: 1 } },
    { name: "pageSize", in: "query", required: false, schema: { type: "integer", minimum: 1, maximum: 100, default: 20 } }
  ];
}

function stableOperation(route, apiPath, audience) {
  const method = route.method.toUpperCase();
  const success = successResponse(route);
  const payload = normalizeStablePayload(apiPath, parseBody(success?.body));
  const responses = {};
  for (const status of statusPolicy(route, apiPath, audience)) {
    if (status === success?.statusCode) {
      if (status === 204) responses[status] = { description: "Completed successfully; no response body." };
      else {
        const examples = { success: { summary: "Successful response", value: payload } };
        if (isListOperation(method, apiPath, payload)) examples.empty = { summary: "Valid empty result", value: emptyExample(payload) };
        responses[status] = { description: "Successful response.", content: { "application/json": { schema: inferSchema(payload), examples } } };
      }
    } else responses[status] = responseRef(status);
  }
  const op = {
    tags: [tagFor(apiPath, audience)],
    operationId: operationId(method, apiPath),
    summary: (route.documentation || `${method} ${apiPath}`)
      .replace(/X-Customer-Actor is derived server-side by the BFF for local Mockoon only\./gi, "Actor context is derived server-side from the authenticated or guest session.")
      .replace(/Mockoon/gi, "local test environment"),
    "x-lifecycle": "stable",
    "x-backlog": backlogFor(apiPath, audience),
    "x-ui-states": isListOperation(method, apiPath, payload) ? ["loading", "success", "empty", "error", "unauthorized", "forbidden"] : ["loading", "success", "error", "unauthorized", "forbidden"],
    "x-test-latency-ms": 1200,
    security: securityFor(apiPath, audience),
    responses
  };
  const params = [...pathParameters(apiPath), ...(isListOperation(method, apiPath, payload) ? listQueryParameters() : [])];
  if (params.length) op.parameters = params;
  const request = requestExamples[`${method} ${apiPath}`];
  if (request && !["GET", "DELETE"].includes(method)) {
    op.requestBody = { required: true, content: { "application/json": { schema: inferSchema(request), example: request } } };
  }
  if (method !== "GET" && ["POST", "PUT", "PATCH", "DELETE"].includes(method)) {
    op.parameters = [...(op.parameters ?? []), { $ref: "#/components/parameters/IdempotencyKey" }];
  }
  return op;
}

function collectStable(sources, audience) {
  const paths = {};
  for (const source of sources) {
    for (const route of source.routes) {
      const apiPath = normalizePath(route.endpoint);
      paths[apiPath] ??= {};
      paths[apiPath][route.method.toLowerCase()] = stableOperation(route, apiPath, audience);
    }
  }
  return Object.fromEntries(Object.entries(paths).sort(([a], [b]) => a.localeCompare(b)));
}

const errorSchema = {
  type: "object",
  required: ["code", "message", "requestId"],
  properties: {
    code: { type: "string", description: "Stable machine-readable error code." },
    message: { type: "string", description: "Safe user-facing message in Vietnamese or the negotiated locale." },
    requestId: { type: "string", description: "Correlation identifier for support and logs." },
    errors: { type: "array", items: { type: "object", required: ["field", "code", "message"], properties: { field: { type: "string" }, code: { type: "string" }, message: { type: "string" } }, additionalProperties: false } },
    retryAfterSeconds: { type: "integer", minimum: 1 },
    details: { type: "object", additionalProperties: true }
  },
  additionalProperties: false
};

function components(audience, paths) {
  const makeResponse = (description, code) => ({ description, content: { "application/json": { schema: { $ref: "#/components/schemas/Error" }, example: { code, message: description, requestId: "req-01J9YAJ6CX8G5E0K7M8N9PQRST", errors: [] } } } });
  const securitySchemes = { bearerAuth: { type: "http", scheme: "bearer", bearerFormat: "JWT" } };
  if (audience === "admin") securitySchemes.adminSession = { type: "apiKey", in: "cookie", name: "admin_session" };
  else securitySchemes.customerSession = { type: "apiKey", in: "cookie", name: "customer_session" };
  const serializedPaths = JSON.stringify(paths);
  if (serializedPaths.includes("webhookSignature")) securitySchemes.webhookSignature = { type: "apiKey", in: "header", name: "X-Signature" };
  if (serializedPaths.includes("guestSession")) securitySchemes.guestSession = { type: "apiKey", in: "cookie", name: "guest_session" };
  if (serializedPaths.includes("guestGrant")) securitySchemes.guestGrant = { type: "apiKey", in: "cookie", name: "guest_order_grant" };
  const responseCatalog = {
    BadRequest: makeResponse("Malformed request or invalid query parameters.", "BAD_REQUEST"),
    Unauthorized: makeResponse("Authentication is missing, invalid, or expired.", "AUTHENTICATION_REQUIRED"),
    Forbidden: makeResponse("Authenticated actor lacks the required permission, capability, ownership, or data scope.", "FORBIDDEN"),
    NotFound: makeResponse("Resource does not exist or is intentionally hidden outside the actor scope.", "RESOURCE_NOT_FOUND"),
    Conflict: makeResponse("State transition, revision, idempotency key, or unique value conflicts with current state.", "STATE_CONFLICT"),
    Gone: makeResponse("Token, challenge, grant, or temporary resource has expired.", "RESOURCE_EXPIRED"),
    UnprocessableEntity: makeResponse("Payload is syntactically valid but violates field or business validation rules.", "VALIDATION_FAILED"),
    TooManyRequests: makeResponse("Rate limit exceeded.", "RATE_LIMITED"),
    InternalError: makeResponse("Unexpected server error.", "INTERNAL_SERVER_ERROR"),
    BadGateway: makeResponse("An upstream response could not be processed.", "UPSTREAM_INVALID_RESPONSE"),
    ServiceUnavailable: makeResponse("A required service is temporarily unavailable.", "SERVICE_UNAVAILABLE")
  };
  const responses = Object.fromEntries(Object.entries(responseCatalog).filter(([name]) => serializedPaths.includes(`#/components/responses/${name}`)));
  return {
    securitySchemes,
    parameters: {
      IdempotencyKey: { name: "Idempotency-Key", in: "header", required: false, description: "Required for retryable create/action operations. Same key and same payload must return the same outcome.", schema: { type: "string", format: "uuid" } }
    },
    schemas: { Error: errorSchema },
    responses
  };
}

function stableSpec(audience, paths) {
  const title = audience === "admin" ? "Internal Employee Web API - Stable" : "Customer Web API - Stable";
  const description = audience === "admin"
    ? "Implementation-ready contract for employee sessions, permission-scoped management views, and approved workbench operations. Authorization MUST use effective permissions and data scopes; role names alone are insufficient."
    : "Implementation-ready contract for Guest, Registered Customer, and verified B2B Customer journeys. Marketplace and Offline are channels, not web login roles.";
  return {
    openapi: "3.1.0",
    info: { title, version: "1.0.0", description, license: { name: "Proprietary", identifier: "LicenseRef-Proprietary" } },
    servers: [{ url: "https://api.omama.vn", description: "Production API gateway" }],
    tags: [...new Set(Object.values(paths).flatMap((item) => Object.values(item).flatMap((op) => op.tags)))].map((name) => ({ name, description: `${name} operations for the ${audience} web application.` })),
    paths,
    components: components(audience, paths)
  };
}

const draftCustomer = [
  ["POST", "/api/v1/media/upload-sessions", "draft", "Create a direct-upload session with type, size and ownership policy.", { purpose: "RETURN_EVIDENCE", fileName: "package.jpg", mediaType: "image/jpeg", sizeBytes: 420000 }, { uploadSessionId: "UPL-01", uploadUrl: "https://storage.example.com/signed", expiresAt: "2026-10-05T09:15:00Z", requiredHeaders: {} }],
  ["POST", "/api/v1/media/upload-sessions/{uploadSessionId}/complete", "draft", "Confirm upload and start malware/content validation.", { checksumSha256: "8f14e45fceea167a5a36dedd4bea2543" }, { fileId: "FILE-01", state: "PROCESSING" }],
  ["GET", "/api/v1/orders/{orderId}/returns/eligibility", "draft", "Evaluate after-sales eligibility from the authoritative order snapshot.", null, { orderId: "ORD-20261015-0042", eligible: true, policyWindowEndsAt: "2026-10-22T00:00:00Z", lines: [] }],
  ["GET", "/api/v1/returns", "draft", "List after-sales cases owned by the current customer.", null, { items: [], total: 0, page: 1, pageSize: 20 }],
  ["POST", "/api/v1/returns/{returnId}/supplements", "draft", "Add requested information or approved evidence to an after-sales case.", { message: "Bo sung anh kien hang", fileIds: ["FILE-01"], idempotencyKey: "64b31cce-a68c-41ef-aa43-99a1b0150ed9" }, { messageId: "MSG-01", occurredAt: "2026-10-05T09:00:00Z" }],
  ["GET", "/api/v1/support/tickets", "draft", "List support tickets visible to the current customer or guest grant.", null, { items: [], total: 0, page: 1, pageSize: 20 }],
  ["POST", "/api/v1/auth/verification-requests", "draft", "Request or resend account contact verification without revealing account existence.", { destination: "an@example.com", channel: "EMAIL" }, { accepted: true, resendAvailableAt: "2026-10-05T09:01:00Z" }],
  ["POST", "/api/v1/auth/verification-requests/confirm", "draft", "Confirm account contact verification.", { verificationToken: "verification-token" }, { verified: true }],
  ["POST", "/api/v1/b2b/quote-requests/{requestId}/messages", "planned", "Continue a NEEDS_INFO exchange while preserving the quote-request history.", { message: "Da bo sung thong tin giao hang", fileIds: [], idempotencyKey: "e501a9cf-ee82-4f8f-87c2-634317f83027" }, { messageId: "B2B-MSG-01", status: "REQUESTED", occurredAt: "2026-10-05T09:00:00Z" }],
  ["GET", "/api/v1/loyalty/transactions", "planned", "List immutable loyalty earning and redemption entries.", null, { items: [], total: 0, page: 1, pageSize: 20 }],
  ["GET", "/api/v1/notifications", "planned", "List customer notifications.", null, { items: [], unreadCount: 0, total: 0 }],
  ["PATCH", "/api/v1/notifications/{notificationId}/read", "planned", "Mark a notification as read idempotently.", { read: true }, { notificationId: "NTF-01", readAt: "2026-10-05T09:00:00Z" }],
  ["GET", "/api/v1/notification-preferences", "planned", "Read customer notification preferences.", null, { email: true, sms: false, push: true, marketing: false }],
  ["PUT", "/api/v1/notification-preferences", "planned", "Replace customer notification preferences.", { email: true, sms: false, push: true, marketing: false }, { email: true, sms: false, push: true, marketing: false }]
];

const draftAdmin = [
  ["POST", "/api/v1/admin/access/employees/invitations", "draft", "Invite an employee account linked to an existing employee profile.", { employeeId: "EMP-0012", email: "employee@example.com", roleAssignments: [] }, { accountId: "ACC-0012", status: "INVITED", revision: 1 }],
  ["PATCH", "/api/v1/admin/access/employees/{employeeId}/status", "draft", "Lock, unlock, disable, or reactivate employee access.", { action: "LOCK", reason: "Security review", expectedRevision: 3 }, { employeeId: "EMP-0012", status: "LOCKED", revision: 4 }],
  ["PUT", "/api/v1/admin/access/employees/{employeeId}/role-assignments", "draft", "Replace effective role assignments using the approved scope taxonomy.", { assignments: [{ roleCode: "PACKING", scope: { type: "ASSIGNED_ONLY" }, effectiveFrom: "2026-10-05T00:00:00Z", effectiveUntil: null }], expectedRevision: 4 }, { employeeId: "EMP-0012", revision: 5 }],
  ["POST", "/api/v1/admin/access/roles", "draft", "Create a custom role after access policy is approved.", { roleCode: "PACKING", roleName: "Packing", permissionCodes: ["PACKING_WORKBENCH_VIEW"] }, { roleCode: "PACKING", status: "ACTIVE", revision: 1 }],
  ["PATCH", "/api/v1/admin/access/roles/{roleCode}", "draft", "Update mutable role metadata with optimistic concurrency.", { roleName: "Packing", description: "Packing team", expectedRevision: 1 }, { roleCode: "PACKING", revision: 2 }],
  ["PUT", "/api/v1/admin/access/roles/{roleCode}/permissions", "draft", "Replace role permissions while protecting privileged roles and the last system administrator.", { permissionCodes: ["PACKING_WORKBENCH_VIEW", "PACKING_TASK_UPDATE"], expectedRevision: 2 }, { roleCode: "PACKING", revision: 3 }],
  ["POST", "/api/v1/admin/catalog/products", "draft", "Create a product with food-information requirements.", { name: "Keo me xung", categoryId: "CAT-01", shortDescription: "Dac san Hue" }, { productId: "PROD-01", saleStatus: "DRAFT", revision: 1 }],
  ["PATCH", "/api/v1/admin/catalog/products/{productId}", "draft", "Update product content with optimistic concurrency.", { shortDescription: "Dac san Hue cao cap", expectedRevision: 1 }, { productId: "PROD-01", revision: 2 }],
  ["POST", "/api/v1/admin/catalog/products/{productId}/skus", "draft", "Create a SKU under a product.", { skuCode: "MX-500", label: "Hop 500g", priceVnd: 345000 }, { skuId: "SKU-01", revision: 1 }],
  ["POST", "/api/v1/admin/orders/{orderId}/actions", "draft", "Apply an authorized order transition with business precondition checks.", { action: "CONFIRM", expectedRevision: 4, idempotencyKey: "17142f65-dad7-44a8-8867-5afabfe2486b" }, { orderId: "ORD-01", status: "CONFIRMED", revision: 5 }],
  ["POST", "/api/v1/admin/packing/tasks/{taskId}/assignments", "draft", "Assign or reassign a packing task within permitted scope.", { employeeId: "EMP-0008", expectedRevision: 2 }, { taskId: "PACK-01", revision: 3 }],
  ["POST", "/api/v1/admin/shipping/shipments/{shipmentId}/actions", "draft", "Create label, confirm handover, retry, or record an exception.", { action: "CONFIRM_HANDOVER", expectedRevision: 5, idempotencyKey: "0f2b46b1-6ed4-4c18-8f98-73b63d3d7f06" }, { shipmentId: "SHP-01", status: "HANDED_OVER", revision: 6 }],
  ["POST", "/api/v1/admin/payments/cases/{caseId}/actions", "draft", "Resolve a reconciliation case under segregation-of-duties policy.", { action: "LINK_TRANSACTION", transactionId: "TXN-01", expectedRevision: 2, idempotencyKey: "ed05640a-c2eb-4ae3-b11d-b3145262f5e5" }, { caseId: "PAYCASE-01", result: "MATCHED", revision: 3 }],
  ["POST", "/api/v1/admin/returns/cases/{caseId}/actions", "draft", "Request information, decide, receive goods, classify, or trigger refund according to the approved state machine.", { action: "REQUEST_INFORMATION", note: "Can them anh", expectedRevision: 2, idempotencyKey: "092564c7-54f2-4fa7-a32c-7ef687775468" }, { caseId: "RET-01", status: "REVIEWING", revision: 3 }],
  ["POST", "/api/v1/admin/support/tickets/{ticketId}/actions", "draft", "Assign, change status, or reopen a support ticket.", { action: "ASSIGN", assigneeId: "EMP-0009", expectedRevision: 3 }, { ticketId: "TKT-01", status: "ASSIGNED", revision: 4 }],
  ["POST", "/api/v1/admin/support/tickets/{ticketId}/messages", "draft", "Send a customer-visible reply through the selected channel.", { content: "Chung toi dang kiem tra", idempotencyKey: "a20beddd-414a-4f08-92df-88ec02fb815b" }, { messageId: "MSG-01", deliveryState: "QUEUED" }],
  ["POST", "/api/v1/admin/support/tickets/{ticketId}/internal-notes", "draft", "Create an internal note that must never be exposed to customers.", { content: "Can doi soat kho" }, { noteId: "NOTE-01", createdAt: "2026-10-05T09:00:00Z" }],
  ["POST", "/api/v1/admin/inventory/approval-requests/{approvalRequestId}/decision", "planned", "Approve or reject a high-risk inventory adjustment with separation of duties.", { decision: "APPROVE", note: "Da doi chieu", expectedRevision: 1 }, { approvalRequestId: "APR-01", status: "APPROVED", revision: 2 }]
];

function draftSpec(audience, definitions) {
  const paths = {};
  for (const [method, apiPath, lifecycle, summary, request, response] of definitions) {
    paths[apiPath] ??= {};
    const explicitSuccess = {
      "POST /api/v1/media/upload-sessions": 201,
      "POST /api/v1/media/upload-sessions/{uploadSessionId}/complete": 202,
      "POST /api/v1/returns/{returnId}/supplements": 201,
      "POST /api/v1/auth/verification-requests": 202,
      "POST /api/v1/auth/verification-requests/confirm": 200,
      "POST /api/v1/b2b/quote-requests/{requestId}/messages": 201,
      "POST /api/v1/admin/access/employees/invitations": 202,
      "POST /api/v1/admin/access/roles": 201,
      "POST /api/v1/admin/catalog/products": 201,
      "POST /api/v1/admin/catalog/products/{productId}/skus": 201,
      "POST /api/v1/admin/payments/cases/{caseId}/actions": 202,
      "POST /api/v1/admin/returns/cases/{caseId}/actions": 202,
      "POST /api/v1/admin/support/tickets/{ticketId}/messages": 201,
      "POST /api/v1/admin/support/tickets/{ticketId}/internal-notes": 201
    };
    const successStatus = explicitSuccess[`${method} ${apiPath}`] ?? 200;
    const responses = {
      [successStatus]: { description: "Proposed successful response.", content: { "application/json": { schema: inferSchema(response), examples: { success: { value: response }, ...(isListOperation(method, apiPath, response) ? { empty: { value: emptyExample(response) } } : {}) } } } }
    };
    const statuses = new Set([400, 422, 500, 503]);
    if (audience === "admin" || !isPublicCustomer(apiPath)) { statuses.add(401); statuses.add(403); }
    if (/\{[^}]+\}/.test(apiPath)) statuses.add(404);
    if (method !== "GET") statuses.add(409);
    if (apiPath.includes("verification-requests")) { statuses.add(410); statuses.add(429); }
    for (const status of [...statuses].sort((a, b) => a - b)) responses[status] = responseRef(status);
    const operation = {
      tags: [tagFor(apiPath, audience)], operationId: operationId(method, apiPath), summary,
      "x-lifecycle": lifecycle, "x-decision-required": true,
      "x-backlog": backlogFor(apiPath, audience),
      "x-ui-states": isListOperation(method, apiPath, response) ? ["loading", "success", "empty", "error", "unauthorized", "forbidden"] : ["loading", "success", "error", "unauthorized", "forbidden"],
      security: securityFor(apiPath, audience), parameters: [...pathParameters(apiPath), ...(isListOperation(method, apiPath, response) ? listQueryParameters() : [])], responses
    };
    if (request) operation.requestBody = { required: true, content: { "application/json": { schema: inferSchema(request), example: request } } };
    if (method !== "GET") operation.parameters.push({ $ref: "#/components/parameters/IdempotencyKey" });
    if (operation.parameters.length === 0) delete operation.parameters;
    paths[apiPath][method.toLowerCase()] = operation;
  }
  return {
    openapi: "3.1.0",
    info: {
      title: audience === "admin" ? "Internal Employee Web API - Draft and Planned" : "Customer Web API - Draft and Planned",
      version: "0.1.0",
      license: { name: "Proprietary", identifier: "LicenseRef-Proprietary" },
      description: "Non-stable proposal for backend review. Every operation requires an explicit product/security decision before implementation compatibility is promised. Draft means a concrete proposal exists; planned means the capability is identified but policy or workflow remains incomplete."
    },
    servers: [{ url: "https://api.omama.vn" }],
    tags: [...new Set(Object.values(paths).flatMap((item) => Object.values(item).flatMap((op) => op.tags)))].map((name) => ({ name, description: `${name} proposed operations for the ${audience} web application.` })),
    paths,
    components: components(audience, paths)
  };
}

function matrixRows(paths) {
  const rows = [];
  for (const [apiPath, item] of Object.entries(paths)) {
    for (const [method, operation] of Object.entries(item)) {
      const statuses = Object.keys(operation.responses).map(Number);
      const success = statuses.filter((status) => status >= 200 && status < 300).join("/");
      const empty = operation["x-ui-states"].includes("empty") ? `${success} (collection empty)` : "N/A";
      const has = (status) => statuses.includes(status) ? String(status) : "—";
      rows.push(`| \`${method.toUpperCase()} ${apiPath}\` | ${operation["x-backlog"].join(", ")} | ${operation["x-lifecycle"]} | ${success} | ${empty} | ${has(400)}/${has(422)} | ${has(401)} | ${has(403)} | ${has(404)} | ${has(409)} | ${has(429)} | ${has(500)}/${has(503)} |`);
    }
  }
  return rows.join("\n");
}

function matrixDocument(audience, stablePaths, draftPaths) {
  const actor = audience === "admin"
    ? "Employee APIs require authentication, effective permission and server-side data-scope enforcement. A hidden menu is not authorization."
    : "Customer APIs distinguish Guest, Registered Customer and verified B2B Customer capabilities. Ownership/grants must be checked server-side.";
  return `# ${audience === "admin" ? "Internal Employee" : "Customer"} API State Matrix\n\n` +
    `Updated: 2026-10-05\n\n` +
    `This matrix and the OpenAPI files are the backend handoff. It is intentionally independent from test-fixture implementation.\n\n` +
    `## State semantics\n\n` +
    `- **Loading:** client state while a request is pending; test with controlled latency. It has no dedicated HTTP status.\n` +
    `- **Empty:** successful \`200\` response with an empty collection and zero totals, never \`404\`.\n` +
    `- **Unauthorized:** \`401\` when credentials/session/grant are missing, invalid or expired.\n` +
    `- **Forbidden:** \`403\` when identity is known but capability, permission, ownership or scope is insufficient.\n` +
    `- **Error:** \`4xx\` is actionable client/business failure; \`5xx\` is server/dependency failure. All errors use the shared \`Error\` schema.\n` +
    `- **Success:** only documented \`2xx\` codes are compatible. Mutations requiring asynchronous processing use \`202\`.\n\n` +
    `${actor}\n\n` +
    `## Stable operations\n\n` +
    `| Operation | Backlog | Lifecycle | Success | Empty | Invalid 400/422 | 401 | 403 | 404 | 409 | 429 | Server 500/503 |\n|---|---|---|---|---|---|---|---|---|---|---|---|\n` +
    `${matrixRows(stablePaths)}\n\n` +
    `## Draft and planned operations\n\n` +
    `These rows are proposals. Backend may refine them, but must record the decision and update the OpenAPI contract before frontend integration.\n\n` +
    `| Operation | Backlog | Lifecycle | Success | Empty | Invalid 400/422 | 401 | 403 | 404 | 409 | 429 | Server 500/503 |\n|---|---|---|---|---|---|---|---|---|---|---|---|\n` +
    `${matrixRows(draftPaths)}\n\n` +
    `## Acceptance rules\n\n` +
    `1. Every list endpoint has both non-empty and empty contract tests.\n` +
    `2. Every protected endpoint has separate \`401\` and \`403\` tests.\n` +
    `3. Every detail endpoint returns \`404\` without leaking whether an out-of-scope resource exists.\n` +
    `4. Every mutation validates \`400/422\`, transition or revision conflict \`409\`, and retry behavior.\n` +
    `5. Every endpoint has controlled latency and \`500/503\` UI tests.\n` +
    `6. Error responses preserve \`code\` and \`requestId\`; clients do not branch on translated \`message\`.\n`;
}

const customerStablePaths = collectStable(customerSources, "customer");
const adminStablePaths = collectStable(adminSources, "admin");
const customerDraftSpec = draftSpec("customer", draftCustomer);
const adminDraftSpec = draftSpec("admin", draftAdmin);

const files = {
  "customer-stable.openapi.yaml": `${yaml(stableSpec("customer", customerStablePaths))}\n`,
  "customer-draft-planned.openapi.yaml": `${yaml(customerDraftSpec)}\n`,
  "admin-stable.openapi.yaml": `${yaml(stableSpec("admin", adminStablePaths))}\n`,
  "admin-draft-planned.openapi.yaml": `${yaml(adminDraftSpec)}\n`,
  "customer-api-state-matrix.md": matrixDocument("customer", customerStablePaths, customerDraftSpec.paths),
  "admin-api-state-matrix.md": matrixDocument("admin", adminStablePaths, adminDraftSpec.paths)
};

for (const [name, content] of Object.entries(files)) fs.writeFileSync(path.join(outputDir, name), content, "utf8");

console.log(`Generated ${Object.keys(files).length} frontend API contract artifacts in ${path.relative(root, outputDir)}.`);
