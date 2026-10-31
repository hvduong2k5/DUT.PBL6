import { createHmac, timingSafeEqual } from "node:crypto";

export const GUEST_ORDER_ACCESS_COOKIE = "oma_order_guest_access";

export interface GuestOrderAccessClaim {
  orderId: string;
  orderNumber: string;
  accessExpiresAt: number;
}

function secret(): string {
  const configured = process.env.ORDER_GUEST_ACCESS_SECRET;
  if (configured) return configured;
  if (process.env.NODE_ENV === "production") throw new Error("ORDER_GUEST_ACCESS_SECRET is required in production.");
  return "oma-local-guest-order-access-secret-change-me";
}

function signature(payload: string): string {
  return createHmac("sha256", secret()).update(payload).digest("base64url");
}

export function encodeGuestOrderAccess(claim: GuestOrderAccessClaim): string {
  const payload = Buffer.from(JSON.stringify(claim), "utf8").toString("base64url");
  return `${payload}.${signature(payload)}`;
}

export function decodeGuestOrderAccess(value: string | undefined): GuestOrderAccessClaim | undefined {
  if (!value) return undefined;
  const [payload, suppliedSignature, extra] = value.split(".");
  if (!payload || !suppliedSignature || extra) return undefined;
  const expected = Buffer.from(signature(payload));
  const supplied = Buffer.from(suppliedSignature);
  if (expected.length !== supplied.length || !timingSafeEqual(expected, supplied)) return undefined;

  try {
    const claim = JSON.parse(Buffer.from(payload, "base64url").toString("utf8")) as GuestOrderAccessClaim;
    if (!claim || typeof claim.orderId !== "string" || !/^[A-Za-z0-9-]{8,100}$/u.test(claim.orderId)) return undefined;
    if (typeof claim.orderNumber !== "string" || claim.orderNumber.length < 8 || claim.orderNumber.length > 100) return undefined;
    if (!Number.isInteger(claim.accessExpiresAt) || claim.accessExpiresAt <= Date.now()) return undefined;
    return claim;
  } catch {
    return undefined;
  }
}

