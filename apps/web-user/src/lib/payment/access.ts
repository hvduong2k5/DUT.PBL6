import { createHmac, timingSafeEqual } from "node:crypto";
import type { PaymentMethod } from "./types";

export const PAYMENT_ACCESS_COOKIE = "oma_payment_access";

export interface PaymentAccessClaim {
  orderId: string;
  orderNumber: string;
  method: PaymentMethod;
  amountVnd: number;
  paymentExpiresAt: string;
  accessExpiresAt: number;
}

function secret(): string {
  const configured = process.env.PAYMENT_ACCESS_SECRET;
  if (configured) return configured;
  if (process.env.NODE_ENV === "production") throw new Error("PAYMENT_ACCESS_SECRET is required in production.");
  return "oma-local-payment-access-secret-change-me";
}

function signature(payload: string): string {
  return createHmac("sha256", secret()).update(payload).digest("base64url");
}

export function encodePaymentAccess(claim: PaymentAccessClaim): string {
  const payload = Buffer.from(JSON.stringify(claim), "utf8").toString("base64url");
  return `${payload}.${signature(payload)}`;
}

export function decodePaymentAccess(value: string | undefined): PaymentAccessClaim | undefined {
  if (!value) return undefined;
  const [payload, suppliedSignature, extra] = value.split(".");
  if (!payload || !suppliedSignature || extra) return undefined;
  const expected = Buffer.from(signature(payload));
  const supplied = Buffer.from(suppliedSignature);
  if (expected.length !== supplied.length || !timingSafeEqual(expected, supplied)) return undefined;
  try {
    const claim = JSON.parse(Buffer.from(payload, "base64url").toString("utf8")) as PaymentAccessClaim;
    if (!claim || typeof claim.orderId !== "string" || typeof claim.orderNumber !== "string") return undefined;
    if (!(["BANK_TRANSFER", "COD"] as string[]).includes(claim.method)) return undefined;
    if (!Number.isInteger(claim.amountVnd) || claim.amountVnd < 0 || !Number.isFinite(Date.parse(claim.paymentExpiresAt))) return undefined;
    if (!Number.isInteger(claim.accessExpiresAt) || claim.accessExpiresAt <= Date.now()) return undefined;
    return claim;
  } catch {
    return undefined;
  }
}

export function paymentAccessCookie(claim: PaymentAccessClaim): string {
  const maxAge = Math.max(1, Math.floor((claim.accessExpiresAt - Date.now()) / 1000));
  const secure = process.env.NODE_ENV === "production" ? "; Secure" : "";
  return `${PAYMENT_ACCESS_COOKIE}=${encodePaymentAccess(claim)}; Path=/api/payments; HttpOnly; SameSite=Lax; Max-Age=${maxAge}${secure}`;
}
