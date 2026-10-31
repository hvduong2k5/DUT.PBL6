import { describe, expect, it } from "vitest";
import { parseSupportRoute, validateCreateSupportTicket, validateSupportMessage } from "./validation";

const valid = { subjectCode: "PRODUCT_ADVICE", title: "Tư vấn bảo quản bánh", message: "Tôi cần được hướng dẫn bảo quản bánh sau khi mở hộp.", requestedPriority: "IMPORTANT", linkedOrderId: "order-mock-001", attachments: [], idempotencyKey: "support-ticket-123456789" };

describe("support validation", () => {
  it("allowlists supported routes", () => {
    expect(parseSupportRoute(["context"]).kind).toBe("context");
    expect(parseSupportRoute(["tickets"]).kind).toBe("tickets");
    expect(parseSupportRoute(["tickets", "ticket-1"]).kind).toBe("ticket");
    expect(parseSupportRoute(["tickets", "ticket-1", "messages"]).kind).toBe("messages");
    expect(parseSupportRoute(["admin", "tickets"]).kind).toBe("invalid");
  });

  it("accepts a valid ticket", () => expect(validateCreateSupportTicket(valid).ok).toBe(true));

  it("validates required fields and attachment policy", () => {
    const result = validateCreateSupportTicket({ ...valid, title: "x", message: "short", attachments: [{ clientReference: "a", fileName: "bad.exe", mediaType: "application/x-msdownload", sizeBytes: 20 * 1024 * 1024 }] });
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.errors.map((item) => item.field)).toEqual(expect.arrayContaining(["title", "message", "attachments.0.mediaType", "attachments.0.sizeBytes"]));
  });

  it("validates guest contact", () => {
    const result = validateCreateSupportTicket({ ...valid, contact: { displayName: "A", email: "bad" } });
    expect(result.ok).toBe(false);
  });

  it("accepts a public reply", () => expect(validateSupportMessage({ message: "Tôi xin bổ sung thêm thông tin cho yêu cầu.", attachments: [], idempotencyKey: "support-message-123456789" }).ok).toBe(true));
});
