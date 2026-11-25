import {afterEach,describe,expect,it,vi} from "vitest";
import {assignmentPayload,loadDispatchSnapshot,validDispatchUuid} from "./company-dispatch";
const id="aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
describe("company dispatch commands",()=>{
  afterEach(()=>vi.unstubAllGlobals());
  it("requires successful history before treating current 404 as unassigned",async()=>{
    const fetcher=vi.fn().mockResolvedValueOnce(Response.json({items:[],total:0})).mockResolvedValueOnce(Response.json({message:"No assignment"},{status:404}));vi.stubGlobal("fetch",fetcher);
    expect(await loadDispatchSnapshot(id)).toEqual({current:null,history:{items:[],total:0}});
    expect(fetcher.mock.calls[0][0]).toContain("/assignment/history");
    fetcher.mockResolvedValueOnce(Response.json({message:"Unknown shipment"},{status:404}));
    await expect(loadDispatchSnapshot(id)).rejects.toThrow("404");expect(fetcher).toHaveBeenCalledTimes(3);
  });
  it("does not turn an outage or inconsistent audit into initial version zero",async()=>{
    const fetcher=vi.fn().mockResolvedValueOnce(Response.json({items:[],total:0})).mockResolvedValueOnce(Response.json({message:"Unavailable"},{status:503}));vi.stubGlobal("fetch",fetcher);
    await expect(loadDispatchSnapshot(id)).rejects.toThrow("503");
    fetcher.mockResolvedValueOnce(Response.json({items:[{}],total:1})).mockResolvedValueOnce(Response.json({message:"No assignment"},{status:404}));
    await expect(loadDispatchSnapshot(id)).rejects.toThrow("chưa nhất quán");
  });
  it("uses explicit assignment version and never accepts actor or shipment fields",()=>{
    expect(assignmentPayload(id,0,"Initial dispatch")).toEqual({driver_id:id,expected_version:0,reason:"Initial dispatch"});
    expect(assignmentPayload(id,2,"Driver change")).toEqual({driver_id:id,expected_version:2,reason:"Driver change"});
  });
  it("rejects nil/invalid UUIDs, stale input shapes and unsafe reasons",()=>{
    expect(validDispatchUuid("00000000-0000-0000-0000-000000000000")).toBe(false);expect(validDispatchUuid("employee-1")).toBe(false);
    for(const [driver,version,reason]of [["bad",0,"Dispatch"],[id,-1,"Dispatch"],[id,1.5,"Dispatch"],[id,Number.MAX_SAFE_INTEGER,"Dispatch"],[id,0," "],[id,0,"Line\nbreak"],[id,0,"x".repeat(501)]] as const)expect(()=>assignmentPayload(driver,version,reason)).toThrow();
  });
});
