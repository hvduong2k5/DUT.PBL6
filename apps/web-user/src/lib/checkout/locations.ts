import type { CheckoutProvince } from "./types";

const CODE_PATTERN = /^[A-Z0-9][A-Z0-9-]{1,31}$/u;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function parseCheckoutLocations(value: unknown): CheckoutProvince[] | undefined {
  if (!isRecord(value) || !Array.isArray(value.provinces) || value.provinces.length < 1 || value.provinces.length > 100) return undefined;
  const provinceCodes = new Set<string>();
  const provinces: CheckoutProvince[] = [];

  for (const item of value.provinces) {
    if (!isRecord(item) || typeof item.provinceCode !== "string" || !CODE_PATTERN.test(item.provinceCode)) return undefined;
    if (typeof item.provinceName !== "string" || item.provinceName.trim() !== item.provinceName || item.provinceName.length < 2 || item.provinceName.length > 100) return undefined;
    if (!Array.isArray(item.wards) || item.wards.length < 1 || item.wards.length > 500 || provinceCodes.has(item.provinceCode)) return undefined;
    provinceCodes.add(item.provinceCode);
    const wardCodes = new Set<string>();
    const wards: CheckoutProvince["wards"] = [];
    for (const ward of item.wards) {
      if (!isRecord(ward) || typeof ward.wardCode !== "string" || !CODE_PATTERN.test(ward.wardCode)) return undefined;
      if (typeof ward.wardName !== "string" || ward.wardName.trim() !== ward.wardName || ward.wardName.length < 2 || ward.wardName.length > 100 || wardCodes.has(ward.wardCode)) return undefined;
      wardCodes.add(ward.wardCode);
      wards.push({ wardCode: ward.wardCode, wardName: ward.wardName });
    }
    provinces.push({ provinceCode: item.provinceCode, provinceName: item.provinceName, wards });
  }
  return provinces;
}

