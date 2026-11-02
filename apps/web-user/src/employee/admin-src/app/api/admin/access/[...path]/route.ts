import { NextRequest } from "next/server";
import { handleAdminAccess } from "../access-handler";

export const dynamic = "force-dynamic";

async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleAdminAccess(request, (await context.params).path);
}

export const GET = route;
