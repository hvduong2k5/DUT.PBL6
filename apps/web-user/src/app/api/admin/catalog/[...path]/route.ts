import { NextRequest } from "next/server";
import { handleAdminCatalog } from "@/features/employee/catalog/server/handler";

export const dynamic = "force-dynamic";

async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleAdminCatalog(request, (await context.params).path);
}

export const GET = route;
export const POST = route;
export const PUT = route;
export const PATCH = route;
export const DELETE = route;
