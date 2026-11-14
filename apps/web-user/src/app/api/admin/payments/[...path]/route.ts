import { NextRequest } from "next/server";
import { handleAdminPayments } from "@/features/employee/payments/server/handler";

export const dynamic = "force-dynamic";
async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) { return handleAdminPayments(request, (await context.params).path); }
export const GET = route;
export const POST = route;
