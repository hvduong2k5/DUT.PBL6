import { NextRequest } from "next/server";
import { handleAdminShipping } from "@/features/employee/shipping/server/handler";
import { handleCompanyShipping } from "@/features/employee/shipping/server/company-handler";

export const dynamic = "force-dynamic";
async function route(request: NextRequest, context: { params: Promise<{ path: string[] }> }) { const path=(await context.params).path; return path[0]==="company" ? handleCompanyShipping(request,path.slice(1)) : handleAdminShipping(request,path); }
export const GET = route;
export const POST = route;
