import { createHash } from "node:crypto";
import { NextRequest, NextResponse } from "next/server";
import { authorizeCustomerCapability, customerAccessFailure } from "@/lib/auth/customer-access-server";
import { GUEST_COOKIE, hasCustomerCredential, withStorefrontAuth, storefrontCustomerCredential, applyStorefrontAuthorization } from "@/lib/auth/storefront-credential";
import { customerCommerceUpstreamUrl } from "@/lib/upstream";
import { hasTrustedOrigin } from "@/lib/auth/request-origin";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";
import { normalizePublicError } from "@/lib/api-contract";
import { isCartMergeResult } from "@/lib/cart/merge";

export const dynamic = "force-dynamic";
function failure(status: number, code: string, message: string) { return NextResponse.json({ code, message, requestId: "BFF-CART-MERGE" }, { status, headers: { "Cache-Control": "no-store" } }); }

async function mergeRequest(request: NextRequest, guestToken?: string, key?: string) {
  return withStorefrontAuth(request, () => resolvedMergeRequest(request, guestToken, key));
}

async function resolvedMergeRequest(request: NextRequest, guestToken?: string, key?: string) {
  if (!hasCustomerCredential(request)) return failure(401,"AUTH_REQUIRED","Vui lòng đăng nhập để gộp giỏ hàng.");
  try { await authorizeCustomerCapability(storefrontCustomerCredential(request),"CART_MANAGE"); }
  catch(cause) { const error=customerAccessFailure(cause);return failure(error.status,error.code,error.message); }
  const headers=new Headers({ Accept:"application/json" });applyStorefrontAuthorization(headers,request);
  if (guestToken) { headers.set("Idempotency-Key",key!);headers.set("Content-Type","application/json"); }
  const base=customerCommerceUpstreamUrl();
  try {
    const response=await fetch(`${base}/cart${guestToken ? "" : `?merge_key=${encodeURIComponent(key!)}`}`,{method:guestToken ? "POST" : "GET",headers,body:guestToken ? JSON.stringify({guest_token:guestToken}) : undefined,cache:"no-store",redirect:"manual"});
    const raw=await response.text();
    const body=JSON.parse(raw) as unknown;
    if (!response.ok) return NextResponse.json(normalizePublicError(body,{code:"CART_MERGE_FAILED",requestId:"BFF-CART-MERGE"}),{status:response.status,headers:{"Cache-Control":"no-store"}});
    if (!isCartMergeResult(body) || body.merge_key !== key) return failure(502,"INVALID_CART_MERGE_RESULT","Kết quả gộp giỏ hàng không hợp lệ.");
    // Preserve persisted int64 Catalog versions; JSON re-encoding through a JS
    // Number would round them and change the result returned on a retry.
    const result=new NextResponse(raw,{headers:{"Content-Type":"application/json","Cache-Control":"no-store"}});
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
