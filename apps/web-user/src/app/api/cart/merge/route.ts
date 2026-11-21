import { createHash } from "node:crypto";
import { NextRequest, NextResponse } from "next/server";
import { authorizeCustomerCapability, customerAccessFailure } from "@/lib/auth/customer-access-server";
import { applyTrustedAuthorization, getTrustedAccessToken } from "@/lib/auth/server-credential";
import { GUEST_COOKIE, hasCustomerCredential } from "@/lib/auth/storefront-credential";
import { hasTrustedOrigin } from "@/lib/auth/request-origin";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";
import { normalizePublicError } from "@/lib/api-contract";
import { isCartMergeResult } from "@/lib/cart/merge";

export const dynamic = "force-dynamic";
function failure(status: number, code: string, message: string) { return NextResponse.json({ code, message, requestId: "BFF-CART-MERGE" }, { status, headers: { "Cache-Control": "no-store" } }); }

async function mergeRequest(request: NextRequest, guestToken?: string, key?: string) {
  if (!hasCustomerCredential(request)) return failure(401,"AUTH_REQUIRED","Vui lòng đăng nhập để gộp giỏ hàng.");
  try { await authorizeCustomerCapability(getTrustedAccessToken(request),"CART_MANAGE"); }
  catch(cause) { const error=customerAccessFailure(cause);return failure(error.status,error.code,error.message); }
  const headers=new Headers({ Accept:"application/json" });applyTrustedAuthorization(headers,request);
  if (guestToken) { headers.set("Idempotency-Key",key!);headers.set("Content-Type","application/json"); }
  const base=(process.env.CUSTOMER_CORE_UPSTREAM_URL ?? "http://127.0.0.1:4010/api/v1").replace(/\/$/u,"");
  try {
    const response=await fetch(`${base}/cart${guestToken ? "" : `?merge_key=${encodeURIComponent(key!)}`}`,{method:guestToken ? "POST" : "GET",headers,body:guestToken ? JSON.stringify({guest_token:guestToken}) : undefined,cache:"no-store",redirect:"manual"});
    const body=await response.json() as unknown;
    if (!response.ok) return NextResponse.json(normalizePublicError(body,{code:"CART_MERGE_FAILED",requestId:"BFF-CART-MERGE"}),{status:response.status,headers:{"Cache-Control":"no-store"}});
    if (!isCartMergeResult(body) || body.merge_key !== key) return failure(502,"INVALID_CART_MERGE_RESULT","Kết quả gộp giỏ hàng không hợp lệ.");
    const result=NextResponse.json(body,{headers:{"Cache-Control":"no-store"}});
    if (guestToken) result.cookies.set(GUEST_COOKIE,"",{httpOnly:true,sameSite:"lax",secure:request.nextUrl.protocol==="https:"||process.env.NODE_ENV==="production"&&process.env.WEB_OIDC_ALLOW_LOCAL_HTTP!=="true",path:"/",maxAge:0});
    return result;
  } catch { return failure(503,"CART_MERGE_UNAVAILABLE","Đã đăng nhập; chưa thể gộp giỏ hàng. Bạn có thể thử lại trong giỏ hàng."); }
}

export async function POST(request: NextRequest) {
  if (!hasTrustedOrigin(request)) return failure(403,"ORIGIN_NOT_ALLOWED","Request origin is not allowed.");
  const token=request.cookies.get(GUEST_COOKIE)?.value;
  // Logging in without a guest cart, or with rollout disabled, is untouched.
  if (!MVP_FEATURE_GATES.guestOrder || process.env.GUEST_CART_MERGE_ENABLED!=="true" || !token) return new NextResponse(null,{status:204,headers:{"Cache-Control":"no-store"}});
  const key=createHash("sha256").update(`cart-merge:${token}`).digest("hex");
  return mergeRequest(request,token,key);
}
export async function GET(request: NextRequest) {
  const key=request.nextUrl.searchParams.get("merge_key");
  if (!key || !/^[0-9a-f]{64}$/u.test(key)) return failure(400,"INVALID_MERGE_KEY","Khóa gộp giỏ hàng không hợp lệ.");
  return mergeRequest(request,undefined,key);
}
