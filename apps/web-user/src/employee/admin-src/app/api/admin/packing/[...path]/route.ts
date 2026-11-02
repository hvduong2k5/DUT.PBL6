import { NextRequest } from "next/server";
import { handleAdminPacking } from "../packing-handler";

export const dynamic = "force-dynamic";

async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) { return handleAdminPacking(request, (await context.params).path); }
export const GET = route;
export const PATCH = route;
export const POST = route;
