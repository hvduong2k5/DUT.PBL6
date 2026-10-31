import { handleOrderRequest } from "../order-handler";
import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleOrderRequest(request, (await context.params).path);
}

export const GET = proxy;
export const POST = proxy;
