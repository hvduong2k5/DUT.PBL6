import { NextRequest } from "next/server";
import { handleReturnRequest } from "../return-handler";

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  return handleReturnRequest(request, path);
}

export const GET = proxy;
export const POST = proxy;
