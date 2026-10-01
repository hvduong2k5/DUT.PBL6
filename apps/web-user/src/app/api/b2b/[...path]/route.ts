import { NextRequest } from "next/server";
import { handleB2BRequest } from "../b2b-handler";

export const dynamic = "force-dynamic";

async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleB2BRequest(request, (await context.params).path);
}

export const GET = route;
export const POST = route;
