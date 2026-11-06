import { describe, expect, it } from "vitest";
import { parseB2BRoute, validateCreateB2BQuoteRequest } from "./validation";

const valid = { organizationId: "org-fpt", purpose: "VIP_CUSTOMER_GIFT", requestedDeliveryDate: "2099-12-20", deliveryProvinceCode: "DA-NANG", deliveryProvinceName: "Thành phố Đà Nẵng", items: [{ skuId: "SKU-001", quantity: 200 }], branding: { engraveLogo: true, customSleeve: false, greetingCard: true, ribbon: false, attachments: [] }, invoiceRequested: true, sampleRequested: true, idempotencyKey: "b2b-request-123456789" };

describe("B2B validation", () => {
  it("routes only the public B2B BFF surface", () => {
    expect(parseB2BRoute(["context"]).kind).toBe("context");
    expect(parseB2BRoute(["quote-requests"]).kind).toBe("requests");
    expect(parseB2BRoute(["quotes", "quote-1"])).toEqual({ kind: "quote", quoteId: "quote-1" });
    expect(parseB2BRoute(["quotes", "quote-1", "accept"])).toEqual({ kind: "accept", quoteId: "quote-1" });
    expect(parseB2BRoute(["admin"]).kind).toBe("invalid");
  });
  it("accepts a complete request and rejects duplicated SKU", () => {
    expect(validateCreateB2BQuoteRequest(valid).ok).toBe(true);
    expect(validateCreateB2BQuoteRequest({ ...valid, items: [valid.items[0], valid.items[0]] }).ok).toBe(false);
  });
  it("rejects an old delivery date and unsafe attachment", () => {
    const result = validateCreateB2BQuoteRequest({ ...valid, requestedDeliveryDate: "2020-01-01", branding: { ...valid.branding, attachments: [{ clientReference: "x", fileName: "x.exe", mediaType: "application/x-msdownload", sizeBytes: 10 }] } });
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.errors.map((item) => item.field)).toEqual(expect.arrayContaining(["requestedDeliveryDate", "branding.attachments"]));
  });
});
