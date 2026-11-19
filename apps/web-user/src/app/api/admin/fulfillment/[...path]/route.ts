import { NextRequest } from "next/server";
import { handleFulfillment } from "@/features/employee/packing/server/fulfillment-handler";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";
async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  return handleFulfillment(request, (await context.params).path);
}
export const GET = route;
export const POST = route;
export const PUT = route;
