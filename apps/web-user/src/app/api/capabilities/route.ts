import { CapabilityProjectionError, fetchCustomerCapabilities } from "@/lib/auth/capability-server";
import { customerAccessFailure, requireCustomerSession } from "@/lib/auth/customer-access-server";
import { withCustomerAuth } from "@/lib/auth/customer-auth";
import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

export async function GET(request: NextRequest) {
  return withCustomerAuth(request, async (credential) => {
    try {
      await requireCustomerSession(credential);
      const projection = await fetchCustomerCapabilities(credential);
      return NextResponse.json(projection);
    } catch (cause) {
      const accessFailure = customerAccessFailure(cause);
      if (accessFailure.status === 401 || accessFailure.status === 403) {
        return NextResponse.json({ code: accessFailure.code, message: accessFailure.message }, { status: accessFailure.status });
      }
      const status = cause instanceof CapabilityProjectionError && cause.status >= 400 && cause.status <= 599 ? cause.status : 503;
      return NextResponse.json({ code: "CAPABILITY_PROJECTION_UNAVAILABLE", message: cause instanceof Error ? cause.message : "Không thể kiểm tra quyền Customer." }, { status });
    }
  });
}
