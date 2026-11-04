import { NextRequest } from "next/server";
import { handleAdminShipping } from "@/features/employee/shipping/server/handler";

export const dynamic = "force-dynamic";
async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) { return handleAdminShipping(request, (await context.params).path); }
export const GET = route;
