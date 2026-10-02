import { describe, expect, it } from "vitest";
import { parseAdminSupportPath } from "./validation";

describe("admin support route validation", () => {
  it("accepts only Ticket list and detail routes", () => {
    expect(parseAdminSupportPath(["tickets"])).toEqual({ kind: "ticket-list" });
    expect(parseAdminSupportPath(["tickets", "ticket-001"])).toEqual({ kind: "ticket-detail", ticketId: "ticket-001" });
    expect(parseAdminSupportPath(["tickets", "ticket-001", "reply"]).kind).toBe("invalid");
  });
});
