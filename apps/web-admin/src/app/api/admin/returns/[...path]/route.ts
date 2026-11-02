import { NextRequest } from "next/server";
import { handleAdminReturns } from "../returns-handler";

export const dynamic = "force-dynamic";
async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) { return handleAdminReturns(request, (await context.params).path); }
export const GET = route;
