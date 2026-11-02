import { NextRequest } from "next/server";
import { handleAdminB2B } from "../b2b-handler";

export const dynamic = "force-dynamic";

async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleAdminB2B(request, (await context.params).path);
}

export const GET = route;
export const POST = route;
