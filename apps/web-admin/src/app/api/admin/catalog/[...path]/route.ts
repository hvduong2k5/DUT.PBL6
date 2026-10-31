import { NextRequest } from "next/server";
import { handleAdminCatalog } from "../catalog-handler";

export const dynamic = "force-dynamic";

async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleAdminCatalog(request, (await context.params).path);
}

export const GET = route;
