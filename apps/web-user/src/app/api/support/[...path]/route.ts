import { NextRequest } from "next/server";
import { handleSupportRequest } from "../support-handler";

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return handleSupportRequest(request, path);
}

export const GET = proxy;
export const POST = proxy;
