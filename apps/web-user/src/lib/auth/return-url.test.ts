import { describe, expect, it } from "vitest";
import { getSafeReturnUrl } from "./return-url";

describe("getSafeReturnUrl", () => {
  it("accepts an internal relative path", () => {
    expect(getSafeReturnUrl("/thanh-toan?step=delivery")).toBe("/thanh-toan?step=delivery");
  });

  it.each(["https://evil.example", "//evil.example", "/%2F%2Fevil.example", "/\\evil.example"])(
    "rejects unsafe destination %s",
    (value) => expect(getSafeReturnUrl(value)).toBe("/")
  );
});
