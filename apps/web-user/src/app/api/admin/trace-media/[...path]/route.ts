import type { NextRequest } from "next/server";
import { handleTraceMedia } from "@/features/employee/traceability/media-server";
export const dynamic = "force-dynamic";
async function handle(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleTraceMedia(request, (await context.params).path);
}
export { handle as GET, handle as POST };
