import { ContractDataError } from "./error";

const STRONG_VERSION_ETAG = /^"([1-9][0-9]*)"$/u;
const DECIMAL_VERSION = /^[1-9][0-9]*$/u;

function positiveVersion(value: number): number {
  if (!Number.isSafeInteger(value) || value < 1) throw new ContractDataError("Version must be a positive safe integer.");
  return value;
}

export function formatStrongVersionEtag(version: number): string {
  return `"${positiveVersion(version)}"`;
}

export function parseStrongVersionEtag(value: string | null | undefined): number | undefined {
  const match = value?.match(STRONG_VERSION_ETAG);
  if (!match) return undefined;
  const parsed = Number(match[1]);
  return Number.isSafeInteger(parsed) ? parsed : undefined;
}

export function formatDecimalVersion(version: number): string {
  return String(positiveVersion(version));
}

export function parseDecimalVersion(value: string | null | undefined): number | undefined {
  if (!value || !DECIMAL_VERSION.test(value)) return undefined;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) ? parsed : undefined;
}

export function readStrongVersionEtag(headers: Headers): number | undefined {
  return parseStrongVersionEtag(headers.get("etag"));
}
