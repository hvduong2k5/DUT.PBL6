import { NextRequest } from "next/server";
import { handleAdminSupport } from "../support-handler";
export const dynamic = "force-dynamic";
async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) { return handleAdminSupport(request, (await context.params).path); }
export const GET = route;
