import { type NextRequest, NextResponse } from "next/server";
import { employeeAuthorizationFailure, requireEmployeeSession } from "@/features/employee/auth/server";

export const dynamic = "force-dynamic";

export async function GET(request: NextRequest) {
  try {
    return NextResponse.json(await requireEmployeeSession(request));
  } catch (cause) {
    const failure = employeeAuthorizationFailure(cause);
    return NextResponse.json({ code: failure.code, message: failure.message }, { status: failure.status });
  }
}
