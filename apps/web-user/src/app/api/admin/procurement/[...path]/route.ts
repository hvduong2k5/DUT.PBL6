import type { NextRequest } from "next/server";
import { handleProcurement } from "@/features/employee/procurement/server/handler";

type Context = { params: Promise<{ path: string[] }> };
async function handle(request: NextRequest, context: Context) {
  return handleProcurement(request, (await context.params).path);
}
export { handle as GET, handle as POST, handle as PUT, handle as PATCH };
