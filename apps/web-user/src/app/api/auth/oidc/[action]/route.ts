import { NextRequest, NextResponse } from "next/server";
import { oidcCallback, oidcLoggedOut, oidcLogin, OidcError } from "@/lib/auth/oidc";
import { usesOidc } from "@/lib/auth/oidc-config";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

export async function GET(request: NextRequest, context: { params: Promise<{ action: string }> }) {
  try {
    if (!usesOidc()) return NextResponse.json({ code: "NOT_FOUND" }, { status: 404 });
    switch ((await context.params).action) {
      case "login": return await oidcLogin(request);
      case "callback": return await oidcCallback(request);
      case "logged-out": return oidcLoggedOut(request);
      default: return NextResponse.json({ code: "NOT_FOUND" }, { status: 404 });
    }
  } catch (error) {
    return NextResponse.json({ code: error instanceof OidcError ? error.code : "OIDC_PROVIDER_UNAVAILABLE", message: "Chưa thể hoàn tất đăng nhập. Vui lòng quay lại trang đăng nhập và thử lại." }, { status: error instanceof OidcError ? error.status : 503, headers: { "Cache-Control": "no-store" } });
  }
}
