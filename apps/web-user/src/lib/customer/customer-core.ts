import type { AddressInput, CustomerAddress, CustomerProfile } from "./types";

export interface CustomerCoreProfile {
  customer_id: string;
  phone_number: string;
  full_name: string;
  email?: string;
  avatar_url?: string;
  membership_tier?: string;
  loyalty_points?: number;
}

export interface CustomerCoreAddress {
  id: string;
  recipient_name: string;
  phone_number: string;
  street_address: string;
  ward: string;
  district: string;
  province: string;
  is_default: boolean;
}

function codeFromName(value: string, level: "province" | "ward") {
  const normalized = value.normalize("NFD").replace(/[\u0300-\u036f]/gu, "").replace(/đ/giu, "d").toUpperCase().replace(/[^A-Z0-9]+/gu, "-").replace(/^-|-$/gu, "");
  if (level === "province" && normalized.includes("HUE")) return "HUE";
  if (level === "ward" && normalized.includes("THUAN-HOA")) return "THUAN-HOA";
  if (level === "province" && normalized.includes("DA-NANG")) return "DA-NANG";
  if (level === "ward") return normalized.replace(/^(?:PHUONG|XA|THI-TRAN)-/u, "");
  return normalized;
}

export function mapCustomerCoreProfile(profile: CustomerCoreProfile): CustomerProfile {
  return {
    id: profile.customer_id,
    fullName: profile.full_name,
    email: profile.email ?? "",
    // The customer API exposes an email address but no verification state.
    emailVerified: false,
    phone: profile.phone_number || null,
    dateOfBirth: null,
    gender: null,
    version: 1,
    updatedAt: new Date().toISOString()
  };
}

export function mapCustomerCoreAddress(address: CustomerCoreAddress): CustomerAddress {
  return {
    id: address.id,
    label: "Địa chỉ nhận hàng",
    type: "HOME",
    recipientName: address.recipient_name,
    recipientPhone: address.phone_number,
    province: { code: codeFromName(address.province, "province"), name: address.province },
    ward: { code: codeFromName(address.ward, "ward"), name: address.ward },
    addressLine: address.street_address,
    deliveryNote: null,
    isDefault: address.is_default,
    version: 1,
    updatedAt: new Date().toISOString()
  };
}

export function toCustomerCoreProfileUpdate(input: { fullName: string; email: string }) {
  return { full_name: input.fullName, email: input.email };
}

export function toCustomerCoreAddressInput(input: AddressInput) {
  return {
    recipient_name: input.recipientName,
    phone_number: input.recipientPhone,
    street_address: input.addressLine,
    ward: input.ward.name,
    district: "",
    province: input.province.name,
    is_default: Boolean(input.isDefault)
  };
}
