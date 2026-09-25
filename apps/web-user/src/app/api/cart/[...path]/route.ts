import { handleCartRequest } from "../cart-handler";
import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleCartRequest(request, (await context.params).path);
}

export const POST = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
