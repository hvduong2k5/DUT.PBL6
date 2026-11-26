import { describe, expect, it } from "vitest";
import { foldLocationText } from "./location-combobox";

describe("foldLocationText", () => {
  it("strips Vietnamese diacritics so unaccented typing matches", () => {
    expect(foldLocationText("Phường Hải Châu")).toBe("phuong hai chau");
    expect(foldLocationText("Thành phố Đà Nẵng")).toBe("thanh pho da nang");
    expect(foldLocationText("Phường Phú Xuân")).toBe("phuong phu xuan");
  });

  it("maps đ/Đ to d and lowercases", () => {
    expect(foldLocationText("Đường Lê Lợi")).toBe("duong le loi");
    expect(foldLocationText("HUẾ")).toBe("hue");
  });
});
