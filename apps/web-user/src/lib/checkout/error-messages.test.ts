import { describe, expect, it } from "vitest";
import { PublicApiError } from "@/lib/api-contract";
import { userErrorCodeMessage, userErrorMessage } from "./error-messages";

function apiError(status: number, code: string, message = "The guest request could not be completed."): PublicApiError {
  return new PublicApiError(status, { code, message, requestId: "req-1" });
}

describe("userErrorMessage", () => {
  it("translates INVALID_ADDRESS instead of leaking the generic server message", () => {
    const message = userErrorMessage(apiError(422, "INVALID_ADDRESS"), "fallback");
    expect(message).toContain("số điện thoại");
    expect(message).not.toContain("guest request");
  });

  it("translates INVALID_ADMINISTRATIVE_UNIT into a ward/province hint", () => {
    const message = userErrorMessage(apiError(422, "INVALID_ADMINISTRATIVE_UNIT"), "fallback");
    expect(message).toContain("Phường/Xã");
  });

  it("translates QUOTE_EXPIRED into a re-quote hint", () => {
    expect(userErrorMessage(apiError(410, "QUOTE_EXPIRED"), "fallback")).toContain("hết hạn");
  });

  it("falls back to the status message for unknown codes", () => {
    expect(userErrorMessage(apiError(503, "SOMETHING_NEW"), "fallback")).toContain("Hệ thống đang bận");
    expect(userErrorMessage(apiError(401, "SOMETHING_NEW"), "fallback")).toContain("đăng nhập");
    expect(userErrorMessage(apiError(422, "SOMETHING_NEW"), "fallback")).toContain("chưa hợp lệ");
  });

  it("uses the caller fallback for non-API errors", () => {
    expect(userErrorMessage(new Error("boom"), "fallback")).toBe("fallback");
    expect(userErrorMessage("boom", "fallback")).toBe("fallback");
  });
});

describe("userErrorCodeMessage", () => {
  it("maps a bare operation error code", () => {
    expect(userErrorCodeMessage("INSUFFICIENT_STOCK")).toContain("tồn kho");
  });

  it("uses the fallback when neither code nor status is known", () => {
    expect(userErrorCodeMessage(undefined, 0, "Checkout thất bại.")).toBe("Checkout thất bại.");
  });
});
