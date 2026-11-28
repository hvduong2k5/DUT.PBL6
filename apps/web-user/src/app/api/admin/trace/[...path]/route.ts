import type { NextRequest } from "next/server";
import { handleTraceability } from "@/features/employee/traceability/server";

export const dynamic = "force-dynamic";
async function handle(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleTraceability(request, (await context.params).path);
}
export { handle as GET, handle as POST, handle as PATCH };
