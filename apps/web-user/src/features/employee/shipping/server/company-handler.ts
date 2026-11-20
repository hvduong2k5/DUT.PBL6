import { NextRequest, NextResponse } from "next/server";
import { usesIdentityBff, usesOidc } from "@/lib/auth/oidc-config";
import { applyTrustedSessionCookie, getTrustedAccessToken } from "@/lib/auth/server-credential";

const uuid = /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/iu;
const record=(v:unknown):v is Record<string,unknown>=>!!v&&typeof v==="object"&&!Array.isArray(v);
function fields(value:unknown,strings:string[],numbers:string[],booleans:string[]=[]):Record<string,unknown>{
  if(!record(value))throw new Error("invalid response");
  const out:Record<string,unknown>={};
  for(const key of strings){if(typeof value[key]!=="string")throw new Error("invalid string");out[key]=value[key];}
  for(const key of numbers){if(!Number.isSafeInteger(value[key])||(value[key] as number)<0)throw new Error("invalid number");out[key]=value[key];}
  for(const key of booleans){if(typeof value[key]!=="boolean")throw new Error("invalid boolean");out[key]=value[key];}return out;
}
function summary(value:unknown){return fields(value,["shipment_id","shipment_number","order_id","package_id","status"],["revision","assignment_version"],["accepted"]);}
function projection(payload:unknown,list:boolean,detail:boolean,accept:boolean){
  if(!record(payload))throw new Error("invalid response");
  if(list){if(!Array.isArray(payload.items)||payload.items.length>100)throw new Error("invalid list");return {...fields(payload,[],["total","limit","offset"]),items:payload.items.map(summary)};}
  if(detail){const value=summary(payload);return {...value,recipient:fields(payload.recipient,["name","phone","street","ward","district","province","country_code"],[]),parcel:fields(payload.parcel,[],["weight_grams","length_cm","width_cm","height_cm"],["fragile"]),...fields(payload,["tracking_code"],[]),handed_over_at:typeof payload.handed_over_at==="string"?payload.handed_over_at:null,estimated_delivery_at:typeof payload.estimated_delivery_at==="string"?payload.estimated_delivery_at:null};}
  return accept?fields(payload,["shipment_id","driver_id","accepted_at"],["assignment_version"]):fields(payload,["shipment_id","actor_id","status","event_id","occurred_at"],["assignment_version","revision"]);
}
function error(status: number, code: string, message: string) {
  return NextResponse.json({ code, message, requestId: `SHIPPING-BFF-${crypto.randomUUID()}` }, { status, headers: { "Cache-Control": "private, no-store" } });
}

// Shipping remains the authority for signed grants, live employee state and ownership.
// Browser Authorization/actor headers and internal tokens are never forwarded.
export async function handleCompanyShipping(request: NextRequest, path: string[]) {
  if (process.env.SHIPPING_COMPANY_BFF_ENABLED !== "true") return error(404, "NOT_FOUND", "Company delivery is not enabled.");
  const list = path.length === 1 && path[0] === "shipments";
  const detail = path.length === 2 && path[0] === "shipments" && uuid.test(path[1]);
  const assignment=path.length===3&&path[0]==="shipments"&&uuid.test(path[1])&&path[2]==="assignment";
  const history=path.length===4&&path[0]==="shipments"&&uuid.test(path[1])&&path[2]==="assignment"&&path[3]==="history";
  const action = (path.length === 3 && path[0] === "shipments" && uuid.test(path[1]) && ["accept", "progress"].includes(path[2])) || (assignment&&request.method==="POST");
  if (!list && !detail && !action&&!assignment&&!history) return error(404, "NOT_FOUND", "Unknown company delivery route.");
  if (request.method !== (action ? "POST" : "GET")) return error(405, "METHOD_NOT_ALLOWED", "Method not allowed.");
  const sessionMode=usesIdentityBff();
  if (!usesOidc()&&!sessionMode) return error(503, "SHIPPING_AUTH_MODE_UNSUPPORTED", "Company delivery requires OIDC or an Identity session.");
  const token = getTrustedAccessToken(request);
  if (!token) return error(401, "AUTH_REQUIRED", "Employee sign-in is required.");
  const query = new URLSearchParams();
  for (const [key, value] of request.nextUrl.searchParams) {
    if (!list || !["limit", "offset"].includes(key) || query.has(key) || !/^\d+$/u.test(value)) return error(400, "INVALID_REQUEST", "Invalid pagination or query override.");
    query.set(key, value);
  }
  const headers = new Headers({ Accept: "application/json" });
  if(sessionMode)applyTrustedSessionCookie(headers,request);else headers.set("Authorization",`Bearer ${token}`);
  let body: string | undefined;
  if (action) {
    // Cookie-authenticated writes require an explicit same-origin browser request.
    if (request.headers.get("origin") !== request.nextUrl.origin || request.headers.get("sec-fetch-site") === "cross-site") return error(403, "UNTRUSTED_ORIGIN", "Same-origin request required.");
    if (request.headers.get("content-type")?.split(";")[0].trim() !== "application/json") return error(415, "UNSUPPORTED_MEDIA_TYPE", "JSON is required.");
    const key = request.headers.get("idempotency-key") ?? "";
    if (!/^[!-~]{16,128}$/u.test(key)) return error(400, "INVALID_IDEMPOTENCY_KEY", "A valid idempotency key is required.");
    const reader=request.body?.getReader();
    const chunks:Uint8Array[]=[];let size=0;
    if(reader){try{while(true){const chunk=await reader.read();if(chunk.done)break;size+=chunk.value.length;if(size>4096){await reader.cancel();return error(413,"PAYLOAD_TOO_LARGE","Action body is too large.");}chunks.push(chunk.value);}}catch{return error(400,"INVALID_REQUEST","Could not read action body.");}finally{reader.releaseLock();}}
    const bytes=new Uint8Array(size);let position=0;for(const chunk of chunks){bytes.set(chunk,position);position+=chunk.length;}body=new TextDecoder().decode(bytes);
    let value: unknown;
    try { value = JSON.parse(body); } catch { return error(400, "INVALID_JSON", "Invalid JSON."); }
    const allowed = assignment ? ["driver_id","expected_version","reason"] : path[2] === "accept" ? ["expected_version"] : ["assignment_version", "expected_revision", "status", "reason"];
    if (!value || typeof value !== "object" || Array.isArray(value) || Object.keys(value).some(key => !allowed.includes(key))) return error(400, "INVALID_REQUEST", "Unknown action fields.");
    headers.set("Content-Type", "application/json"); headers.set("Idempotency-Key", key);
  }
  try {
    const base = new URL((sessionMode?process.env.IDENTITY_UPSTREAM_URL:process.env.SHIPPING_COMPANY_UPSTREAM_URL) ?? "");
    if (base.username || base.password || base.search || base.hash || (sessionMode?!["/","/api/v1","/api/v1/"].includes(base.pathname):base.pathname!=="/") || (base.protocol !== "https:" && !(base.protocol === "http:" && (sessionMode || (process.env.NODE_ENV !== "production" && ["localhost", "127.0.0.1", "[::1]"].includes(base.hostname)))))) throw new Error("invalid upstream");
    const target = history?`shipments/${path[1]}/assignment/history`:assignment?`shipments/${path[1]}/assignment`:action ? `shipments/${path[1]}/${path[2]}` : `workbench/shipments${detail ? `/${path[1]}` : ""}`;
    const response = await fetch(`${base.origin}${sessionMode?"/api/v1/admin/shipping-api":"/api/v2/admin/shipping"}/${target}${query.size ? `?${query}` : ""}`, { method: request.method, headers, body, cache: "no-store", redirect: "manual", signal: AbortSignal.timeout(10000) });
    if (response.status >= 300 && response.status < 400) return error(502, "SHIPPING_INVALID_RESPONSE", "Shipping redirect rejected.");
    if (!response.headers.get("content-type")?.includes("application/json")) return error(502, "SHIPPING_INVALID_RESPONSE", "Invalid Shipping response.");
    const payload: unknown = await response.json();
    if (!response.ok) {
      // Never relay upstream diagnostics, tokens or stack traces to the browser.
      const codes: Record<number, string> = { 400: "INVALID_REQUEST", 401: "AUTH_REQUIRED", 403: "FORBIDDEN", 404: "NOT_FOUND", 409: "CONFLICT", 413: "PAYLOAD_TOO_LARGE", 415: "UNSUPPORTED_MEDIA_TYPE", 422: "VALIDATION_FAILED", 503: "SHIPPING_UNAVAILABLE" };
      return error(response.status >= 400 && response.status <= 599 ? response.status : 502, codes[response.status] ?? "SHIPPING_UNAVAILABLE", "Shipping could not complete this request. Refresh after a conflict; retry network failures with the same key.");
    }
    try{
      let result:unknown;
      if(assignment)result=assignmentProjection(payload);
      else if(history){if(!record(payload)||!Array.isArray(payload.items))throw new Error("invalid history");result={...fields(payload,[],["total"]),items:payload.items.map(item=>{const out=assignmentProjection(item);if(!record(item)||(item.previous_driver_id!==null&&typeof item.previous_driver_id!=="string"))throw new Error("invalid previous driver");return {...out,...fields(item,["event_id"],[]),previous_driver_id:item.previous_driver_id};})};}
      else result=projection(payload,list,detail,path[2]==="accept");
      return NextResponse.json(result, { headers: { "Cache-Control": "private, no-store" } });}
    catch{return error(502,"SHIPPING_INVALID_RESPONSE","Shipping returned an invalid projection.");}
  } catch { return error(503, "SHIPPING_UNAVAILABLE", "Shipping is temporarily unavailable."); }
}

function assignmentProjection(value:unknown){return fields(value,["shipment_id","driver_id","assigned_by","reason","assigned_at"],["version"]);}
