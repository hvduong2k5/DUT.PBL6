import { createHash } from "node:crypto";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";

let route: typeof import("@/app/api/cart/merge/route");
let access: typeof import("@/lib/auth/customer-access-server");
const customer = "10000000-0000-4000-8000-000000000053";
const key = createHash("sha256").update("cart-merge:guest-token").digest("hex");
const result = { merge_key: key, cart: { revision: 2, items: [{ item_id: "SKU-A", sku_code: "SKU-A", quantity: 5 }] }, items: [{ sku_code: "SKU-A", status: "merged", requested_quantity: 5, quantity: 5 }], merged_at: "2026-11-21T12:00:00Z" };
const request = (guest = true) => new NextRequest("http://localhost/api/cart/merge", { method: "POST", headers: { Origin: "http://localhost", Cookie: `OMAMA_SESSION=${customer}${guest ? "; oma_guest_session=guest-token" : ""}` }, body: "{}" });
beforeEach(async () => {
  vi.resetModules();vi.doUnmock("@/lib/auth/customer-access-server");
  vi.doMock("@/lib/mvp-feature-gates",()=>({MVP_FEATURE_GATES:{guestOrder:true}}));
  vi.stubEnv("WEB_AUTH_MODE","identity-bff");vi.stubEnv("WEB_PUBLIC_URL","http://localhost");vi.stubEnv("WEB_OIDC_ALLOW_LOCAL_HTTP","true");vi.stubEnv("GUEST_CART_MERGE_ENABLED","true");vi.stubEnv("CUSTOMER_CORE_UPSTREAM_URL","http://gateway/api/v1/customer-api");
  access=await import("@/lib/auth/customer-access-server");
  vi.spyOn(access,"authorizeCustomerCapability").mockResolvedValue({} as Awaited<ReturnType<typeof access.authorizeCustomerCapability>>);
  route=await import("@/app/api/cart/merge/route");
});
afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs();vi.restoreAllMocks();vi.doUnmock("@/lib/mvp-feature-gates");});

it("leaves login without a guest cookie untouched",async()=>{
  const fetcher=vi.fn();vi.stubGlobal("fetch",fetcher);
  expect((await route.POST(request(false))).status).toBe(204);
  expect(fetcher).not.toHaveBeenCalled();expect(access.authorizeCustomerCapability).not.toHaveBeenCalled();
});
it("keeps Customer auth and source auth separate and clears only the guest cookie on success",async()=>{
  const fetcher=vi.fn<(url:string,options:{headers:Headers;body:string})=>Response>().mockImplementation(()=>Response.json(result));vi.stubGlobal("fetch",fetcher);
  const response=await route.POST(request());expect(response.status).toBe(200);
  const options=fetcher.mock.calls[0][1];expect(fetcher.mock.calls[0][0]).toBe("http://gateway/api/v1/customer-api/cart");
  expect(options.headers.get("Cookie")).toBe(`OMAMA_SESSION=${customer}`);expect(options.headers.has("Authorization")).toBe(false);
  expect(JSON.parse(options.body)).toEqual({guest_token:"guest-token"});expect(options.headers.get("Idempotency-Key")).toBe(key);
  expect(response.headers.get("set-cookie")).toContain("oma_guest_session=");expect(response.headers.get("set-cookie")).not.toContain("OMAMA_SESSION");
  expect(JSON.stringify(await response.json())).not.toContain("guest-token");
});
it("preserves both login and source cookies when the merge fails",async()=>{
  vi.stubGlobal("fetch",vi.fn().mockResolvedValue(Response.json({code:"CART_MERGE_UNAVAILABLE",message:"Retry",requestId:"test"},{status:503})));
  const response=await route.POST(request());expect(response.status).toBe(503);expect(response.headers.has("set-cookie")).toBe(false);
});
it("preserves persisted int64 Catalog versions in the response bytes",async()=>{
  const raw=JSON.stringify(result).replace('"quantity":5}', '"quantity":5,"catalog_version":560372688932839723}');
  vi.stubGlobal("fetch",vi.fn().mockResolvedValue(new Response(raw,{headers:{"Content-Type":"application/json"}})));
  const response=await route.POST(request());
  expect(response.status).toBe(200);
  expect(await response.text()).toBe(raw);
});
it("does not change the merge key on retries and never falls back from invalid Customer auth",async()=>{
  const fetcher=vi.fn<(url:string,options:{headers:Headers;body:string})=>Response>().mockImplementation(()=>Response.json(result));vi.stubGlobal("fetch",fetcher);
  await route.POST(request());await route.POST(request());
  expect(fetcher.mock.calls[0][1].headers.get("Idempotency-Key")).toBe(fetcher.mock.calls[1][1].headers.get("Idempotency-Key"));
  vi.mocked(access.authorizeCustomerCapability).mockRejectedValueOnce(new access.CustomerAccessError(401,"AUTH_REQUIRED","Expired"));
  expect((await route.POST(request())).status).toBe(401);expect(fetcher).toHaveBeenCalledTimes(2);
});
