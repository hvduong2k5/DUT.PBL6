import { describe, expect, it } from "vitest";
import { parseReturnRoute, validateCreateReturnCase, validateSupplement } from "./validation";

const validCreate = {
  orderId: "order-mock-20260926-001",
  lines: [{ lineId: "line-order-gift", quantity: 1 }],
  reasonCode: "DAMAGED_IN_TRANSIT",
  preferredResolution: "REPLACEMENT",
  details: "Góc hộp bị móp khi nhận hàng.",
  evidence: [],
  idempotencyKey: "return-case-550e8400-e29b-41d4-a716-446655440000"
};

describe("Return route validation", () => {
  it("allowlists supported operations", () => {
    expect(parseReturnRoute("GET", ["orders", "order-mock-20260926-001", "eligibility"], new URLSearchParams()).operation).toBe("eligibility");
    expect(parseReturnRoute("POST", [], new URLSearchParams()).operation).toBe("create");
    expect(parseReturnRoute("GET", ["return-case-mock-001"], new URLSearchParams()).operation).toBe("detail");
    expect(parseReturnRoute("POST", ["return-case-mock-001", "supplements"], new URLSearchParams()).operation).toBe("supplement");
  });

  it("rejects unsupported query and path", () => {
    expect(parseReturnRoute("GET", ["return-case-mock-001"], new URLSearchParams("owner=x")).error?.field).toBe("query");
    expect(parseReturnRoute("DELETE", ["return-case-mock-001"], new URLSearchParams()).error?.field).toBe("path");
  });
});

describe("Return body validation", () => {
  it("accepts a bounded partial-line request", () => {
    expect(validateCreateReturnCase(validCreate).data?.lines[0]).toEqual({ lineId: "line-order-gift", quantity: 1 });
  });

  it("rejects duplicate lines and oversized evidence", () => {
    const result = validateCreateReturnCase({ ...validCreate, lines: [...validCreate.lines, ...validCreate.lines], evidence: [{ clientReference: "evidence-001", fileName: "large.mp4", mediaType: "video/mp4", sizeBytes: 16 * 1024 * 1024 }] });
    expect(result.errors.some((error) => error.field.startsWith("lines."))).toBe(true);
    expect(result.errors.some((error) => error.field.startsWith("evidence."))).toBe(true);
  });

  it("validates supplement idempotency and length", () => {
    expect(validateSupplement({ message: "Bổ sung giờ nhận hàng.", idempotencyKey: "return-supplement-550e8400-e29b-41d4-a716-446655440000" }).data?.message).toContain("Bổ sung");
    expect(validateSupplement({ message: "", idempotencyKey: "bad" }).errors).toHaveLength(2);
  });
});
