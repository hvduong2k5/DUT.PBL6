import { CapabilityProjectionError, fetchCustomerCapabilities } from "@/lib/auth/capability-server";
import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

export async function GET(request: NextRequest) {
  try {
    const projection = await fetchCustomerCapabilities(request.cookies.get("oma_access_token")?.value);
    return NextResponse.json(projection);
  } catch (cause) {
    const status = cause instanceof CapabilityProjectionError && cause.status >= 400 && cause.status <= 599 ? cause.status : 503;
    return NextResponse.json({ code: "CAPABILITY_PROJECTION_UNAVAILABLE", message: cause instanceof Error ? cause.message : "Không thể kiểm tra quyền Customer." }, { status });
  }
}
