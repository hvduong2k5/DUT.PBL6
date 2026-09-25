import { handleCheckoutRequest } from "../checkout-handler";
import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleCheckoutRequest(request, (await context.params).path);
}

export const POST = proxy;
