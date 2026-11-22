import type { NextRequest } from "next/server";
import { recommendationProxy } from "../recommendation-handler";

export const dynamic = "force-dynamic";

async function handler(request: NextRequest, context: { params: Promise<{ path?: string[] }> }) {
  return recommendationProxy(request, (await context.params).path ?? []);
}

export const GET = handler;
export const PUT = handler;
export const POST = handler;
