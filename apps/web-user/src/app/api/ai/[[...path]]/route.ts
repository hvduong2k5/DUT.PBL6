import type { NextRequest } from "next/server";
import { aiChatProxy } from "../ai-chat-handler";

export const dynamic = "force-dynamic";

async function handler(request: NextRequest, context: { params: Promise<{ path?: string[] }> }) {
  return aiChatProxy(request, (await context.params).path ?? []);
}

export const GET = handler;
export const PUT = handler;
export const POST = handler;
