import { NextRequest, NextResponse } from "next/server";
import { handleAdminAccess } from "@admin/app/api/admin/access/access-handler";
import { handleAdminB2B } from "@admin/app/api/admin/b2b/b2b-handler";
import { handleAdminCatalog } from "@admin/app/api/admin/catalog/catalog-handler";
import { handleAdminInventory } from "@admin/app/api/admin/inventory/inventory-handler";
import { handleAdminOrders } from "@admin/app/api/admin/orders/orders-handler";
import { handleAdminPacking } from "@admin/app/api/admin/packing/packing-handler";
import { handleAdminPayments } from "@admin/app/api/admin/payments/payments-handler";
import { handleAdminReturns } from "@admin/app/api/admin/returns/returns-handler";
import { handleAdminSession } from "@admin/app/api/admin/session/route";
import { handleAdminShipping } from "@admin/app/api/admin/shipping/shipping-handler";
import { handleAdminSupport } from "@admin/app/api/admin/support/support-handler";

export const dynamic = "force-dynamic";

async function route(request: NextRequest, context: { params: Promise<{ path?: string[] }> }) {
  const [area, ...path] = (await context.params).path ?? [];
  switch (area) {
    case "access": return handleAdminAccess(request, path);
    case "b2b": return handleAdminB2B(request, path);
    case "catalog": return handleAdminCatalog(request, path);
    case "inventory": return handleAdminInventory(request, path);
    case "orders": return handleAdminOrders(request, path);
    case "packing": return handleAdminPacking(request, path);
    case "payments": return handleAdminPayments(request, path);
    case "returns": return handleAdminReturns(request, path);
    case "session": return request.method === "GET" ? handleAdminSession(request) : NextResponse.json({ code: "METHOD_NOT_ALLOWED" }, { status: 405 });
    case "shipping": return handleAdminShipping(request, path);
    case "support": return handleAdminSupport(request, path);
    default: return NextResponse.json({ code: "NOT_FOUND" }, { status: 404 });
  }
}

export const GET = route;
export const POST = route;
export const PATCH = route;
export const PUT = route;
export const DELETE = route;
