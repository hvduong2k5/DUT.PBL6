import { describe, expect, it } from "vitest";
import { getPasswordChecks, isPasswordValid, passwordCodePointLength } from "./validation";

describe("approved password policy", () => {
  it("accepts a password containing every approved class", () => {
    expect(isPasswordValid("DiSanCoDoHue#2026")).toBe(true);
  });

  it("rejects whitespace as a replacement for a special character", () => {
    const special = getPasswordChecks("DiSanCoDoHue 2026").find(
      (check) => check.code === "PASSWORD_MISSING_SPECIAL"
    );
    expect(special?.valid).toBe(false);
  });

  it("counts Unicode code points instead of UTF-16 code units", () => {
    expect(passwordCodePointLength("A😀b1!abcdefghij")).toBe(15);
  });
});
