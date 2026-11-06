import { ContractDataError } from "./error";

const VND_FORMATTER = new Intl.NumberFormat("vi-VN", {
  style: "currency",
  currency: "VND",
  maximumFractionDigits: 0
});

function toSafeInteger(value: unknown, field: string): number {
  const integer = typeof value === "number"
    ? value
    : typeof value === "string" && /^(?:0|[1-9][0-9]*)$/u.test(value)
      ? Number(value)
      : Number.NaN;
  if (!Number.isSafeInteger(integer) || integer < 0) {
    throw new ContractDataError(`${field} must be a non-negative safe integer.`);
  }
  return integer;
}

export function parseVndMoney(value: unknown): number {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new ContractDataError("Money must be an object.");
  }
  const money = value as Record<string, unknown>;
  if (money.currency_code !== "VND") throw new ContractDataError("Money currency must be VND.");
  if (money.nanos !== 0) throw new ContractDataError("VND Money nanos must be zero.");
  return toSafeInteger(money.units, "Money.units");
}

export function parseVndAmount(value: unknown): number {
  return toSafeInteger(value, "VND amount");
}

export function formatVnd(value: number): string {
  if (!Number.isSafeInteger(value) || value < 0) throw new ContractDataError("VND amount must be a non-negative safe integer.");
  return VND_FORMATTER.format(value);
}
