import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { handleCompanyShipping } from "./company-handler";
const id="aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
function request(method="GET",body?:string,origin="http://localhost:3000"){
  return new NextRequest("http://localhost:3000/api/admin/shipping/company/shipments",{method,headers:{Cookie:"oma_access_token=trusted-staff; unrelated=private",Authorization:"Bearer forged","X-Internal-Token":"forged","X-Actor-ID":"forged",Origin:origin,"Content-Type":"application/json","Idempotency-Key":"action-key-0000001"},body});
}
describe("company Shipping BFF",()=>{
  beforeEach(()=>{vi.stubEnv("WEB_AUTH_MODE","oidc");vi.stubEnv("SHIPPING_COMPANY_BFF_ENABLED","true");vi.stubEnv("SHIPPING_COMPANY_UPSTREAM_URL","http://127.0.0.1:8012");});
  afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs();});
  it("forwards only trusted bearer to exact owned read routes",async()=>{
    const fetcher=vi.fn().mockResolvedValue(Response.json({items:[],total:0,limit:50,offset:0}));vi.stubGlobal("fetch",fetcher);
    const response=await handleCompanyShipping(request(),["shipments"]);
    expect(response.status).toBe(200);expect(response.headers.get("cache-control")).toBe("private, no-store");
    const [url,init]=fetcher.mock.calls[0];expect(url).toBe("http://127.0.0.1:8012/api/v2/admin/shipping/workbench/shipments");
    expect(init.headers.get("Authorization")).toBe("Bearer trusted-staff");
    for(const key of ["Cookie","X-Actor-ID","X-Internal-Token"])expect(init.headers.get(key)).toBeNull();
    expect(init.redirect).toBe("manual");
  });
  it("preserves action payload/key without actor or token injection",async()=>{
    const fetcher=vi.fn().mockResolvedValue(Response.json({shipment_id:id,driver_id:id,assignment_version:1,accepted_at:"2026-11-20T10:00:00Z",access_token:"must-not-leak"}));vi.stubGlobal("fetch",fetcher);
    const response=await handleCompanyShipping(request("POST",'{"expected_version":1}'),["shipments",id,"accept"]);expect(response.status).toBe(200);expect(await response.text()).not.toContain("must-not-leak");
    const [url,init]=fetcher.mock.calls[0];expect(url).toBe(`http://127.0.0.1:8012/api/v2/admin/shipping/shipments/${id}/accept`);expect(init.headers.get("Idempotency-Key")).toBe("action-key-0000001");expect(init.body).toBe('{"expected_version":1}');
  });
  it("fails closed for disabled, session mode, missing credential and forged paths",async()=>{
    const fetcher=vi.fn();vi.stubGlobal("fetch",fetcher);
    vi.stubEnv("SHIPPING_COMPANY_BFF_ENABLED","false");expect((await handleCompanyShipping(request(),["shipments"])).status).toBe(404);
    vi.stubEnv("SHIPPING_COMPANY_BFF_ENABLED","true");vi.stubEnv("WEB_AUTH_MODE","identity-bff");expect((await handleCompanyShipping(request(),["shipments"])).status).toBe(503);
    vi.stubEnv("WEB_AUTH_MODE","oidc");expect((await handleCompanyShipping(new NextRequest("http://localhost:3000",{headers:{Authorization:"Bearer forged"}}),["shipments"])).status).toBe(401);
    expect((await handleCompanyShipping(request(),["shipments","../private"])).status).toBe(404);expect(fetcher).not.toHaveBeenCalled();
  });
  it("rejects cross-origin, actor overrides, oversized and malformed bodies",async()=>{
    const fetcher=vi.fn();vi.stubGlobal("fetch",fetcher);
    expect((await handleCompanyShipping(request("POST",'{"expected_version":1}',"https://evil.test"),["shipments",id,"accept"])).status).toBe(403);
    expect((await handleCompanyShipping(request("POST",'{"expected_version":1,"actor_id":"spoof"}'),["shipments",id,"accept"])).status).toBe(400);
    expect((await handleCompanyShipping(request("POST","not-json"),["shipments",id,"accept"])).status).toBe(400);
    expect((await handleCompanyShipping(request("POST","x".repeat(4097)),["shipments",id,"accept"])).status).toBe(413);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("preserves conflict/auth status without relaying private diagnostics or redirects",async()=>{
    const fetcher=vi.fn().mockResolvedValue(Response.json({message:"secret diagnostic"},{status:409}));vi.stubGlobal("fetch",fetcher);
    const response=await handleCompanyShipping(request("POST",'{"expected_version":1}'),["shipments",id,"accept"]);expect(response.status).toBe(409);expect(await response.text()).not.toContain("secret diagnostic");
    fetcher.mockResolvedValue(new Response(null,{status:302,headers:{Location:"https://evil.test"}}));expect((await handleCompanyShipping(request(),["shipments"])).status).toBe(502);
  });
});
