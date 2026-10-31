import { describe, expect, it } from "vitest";
import { parseCheckoutLocations } from "./locations";

describe("checkout location projection", () => {
  it("accepts only a two-level province and ward tree", () => {
    expect(parseCheckoutLocations({ provinces: [{
      provinceCode: "HUE", provinceName: "Thành phố Huế",
      wards: [{ wardCode: "THUAN-HOA", wardName: "Phường Thuận Hòa" }]
    }] })?.[0].wards[0].wardCode).toBe("THUAN-HOA");
  });

  it("rejects a legacy district level", () => {
    expect(parseCheckoutLocations({ provinces: [{
      provinceCode: "HUE", provinceName: "Thành phố Huế",
      districts: [{ districtCode: "HUE-CENTER", wards: [] }]
    }] })).toBeUndefined();
  });
});
