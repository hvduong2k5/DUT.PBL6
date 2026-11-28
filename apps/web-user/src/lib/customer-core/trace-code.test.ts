import { describe, expect, it } from "vitest";
import { traceCode } from "./trace-code";
describe("label lookup", () => {
  it("accepts tokens and exact storefront links", () => {
    expect(traceCode(" abc_DEF-123 ", [])).toBe("abc_DEF-123");
    expect(traceCode("https://shop.example.test/trace/abc_DEF-123", ["https://shop.example.test"])).toBe("abc_DEF-123");
  });
  it("rejects arbitrary destinations, path traversal and query ambiguity", () => {
    for (const input of ["", "../../account", "https://attacker.test/trace/token", "https://user:password@shop.example.test/trace/token", "https://shop.example.test/trace/token?next=evil", "https://shop.example.test/trace/token#x"]) expect(traceCode(input, ["https://shop.example.test"])).toBeUndefined();
  });
});
