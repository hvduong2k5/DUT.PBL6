import { CapabilityProjectionError, fetchCustomerCapabilities } from "@/lib/auth/capability-server";
import { customerAccessFailure, requireCustomerSession } from "@/lib/auth/customer-access-server";
import { getTrustedAccessToken } from "@/lib/auth/server-credential";
import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

export async function GET(request: NextRequest) {
  try {
    const accessToken = getTrustedAccessToken(request);
    await requireCustomerSession(accessToken);
    const projection = await fetchCustomerCapabilities(accessToken);
    return NextResponse.json(projection);
  } catch (cause) {
    const accessFailure = customerAccessFailure(cause);
    if (accessFailure.status === 401 || accessFailure.status === 403) {
      return NextResponse.json({ code: accessFailure.code, message: accessFailure.message }, { status: accessFailure.status });
    }
    const status = cause instanceof CapabilityProjectionError && cause.status >= 400 && cause.status <= 599 ? cause.status : 503;
    return NextResponse.json({ code: "CAPABILITY_PROJECTION_UNAVAILABLE", message: cause instanceof Error ? cause.message : "Không thể kiểm tra quyền Customer." }, { status });
  }
}
