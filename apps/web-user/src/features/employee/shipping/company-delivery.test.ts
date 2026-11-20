import { describe,expect,it } from "vitest";
import { nextCompanyStatuses } from "./company-delivery";
describe("company driver actions",()=>{
  it("does not manufacture warehouse handover or change terminal states",()=>{for(const status of ["CREATED","READY_FOR_HANDOVER","DELIVERED","RETURNED","CANCELLED","unknown"])expect(nextCompanyStatuses(status)).toEqual([]);});
  it("matches backend stages including failed delivery retry/return",()=>{expect(nextCompanyStatuses("HANDED_OVER")).toEqual(["IN_TRANSIT"]);expect(nextCompanyStatuses("IN_TRANSIT")).toEqual(["OUT_FOR_DELIVERY"]);expect(nextCompanyStatuses("OUT_FOR_DELIVERY")).toEqual(["DELIVERED","DELIVERY_FAILED"]);expect(nextCompanyStatuses("DELIVERY_FAILED")).toEqual(["OUT_FOR_DELIVERY","RETURNING"]);expect(nextCompanyStatuses("RETURNING")).toEqual(["RETURNED"]);});
});
