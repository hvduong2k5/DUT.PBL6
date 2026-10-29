import { ContractDataError, parseStrongVersionEtag } from "@/lib/api-contract";
import type {
  AddressUpdate,
  AddressWrite,
  Profile,
  ProfileUpdate,
  ShippingAddress
} from "@/lib/api-contract";
import type {
  AddressInput,
  AddressList,
  AddressType,
  CustomerAddress,
  CustomerProfile,
  Gender,
  ProfileInput
} from "./types";

function requiredText(value: unknown, field: string): string {
  if (typeof value !== "string" || !value.trim()) throw new ContractDataError(`${field} is required.`);
  return value;
}

function positiveVersion(value: unknown, etag?: string | null): number {
  const fromEtag = parseStrongVersionEtag(etag);
  if (fromEtag) return fromEtag;
  if (typeof value === "number" && Number.isSafeInteger(value) && value > 0) return value;
  throw new ContractDataError("A positive resource version is required.");
}

function profileGender(value: unknown): Gender | null {
  return value === "MALE" || value === "FEMALE" || value === "OTHER" || value === "PREFER_NOT_TO_SAY" ? value : null;
}

function addressType(value: unknown): AddressType {
  if (value === "OFFICE") return "OFFICE";
  if (value === "GIFT" || value === "GIFT_RECIPIENT") return "GIFT";
  if (value === "OTHER") return "OTHER";
  return "HOME";
}

function addressLabel(type: AddressType): string {
  if (type === "OFFICE") return "Văn phòng";
  if (type === "GIFT") return "Người nhận quà";
  if (type === "OTHER") return "Địa chỉ khác";
  return "Nhà riêng";
}

export function mapMvpProfile(profile: Profile, etag?: string | null): CustomerProfile {
  const updatedAt = profile.updated_at ?? profile.created_at;
  return {
    id: requiredText(profile.id ?? profile.user_id, "Profile.id"),
    fullName: requiredText(profile.full_name, "Profile.full_name"),
    email: typeof profile.email === "string" ? profile.email : "",
    emailVerified: false,
    phone: typeof profile.phone_number === "string" && profile.phone_number ? profile.phone_number : null,
    dateOfBirth: typeof profile.date_of_birth === "string" ? profile.date_of_birth.slice(0, 10) : null,
    gender: profileGender(profile.gender),
    version: positiveVersion(profile.version, etag),
    updatedAt: requiredText(updatedAt, "Profile.updated_at")
  };
}

export function toMvpProfileUpdate(input: ProfileInput): ProfileUpdate {
  return {
    full_name: input.fullName.trim(),
    phone_number: input.phone ? input.phone.trim() : ""
  };
}

export function mapMvpAddress(address: ShippingAddress, etag?: string | null): CustomerAddress {
  const type = addressType(address.label);
  return {
    id: requiredText(address.id, "Address.id"),
    label: addressLabel(type),
    type,
    recipientName: requiredText(address.recipient_name, "Address.recipient_name"),
    recipientPhone: requiredText(address.phone_number, "Address.phone_number"),
    province: {
      code: requiredText(address.province_code, "Address.province_code"),
      name: requiredText(address.province_name, "Address.province_name")
    },
    ward: {
      code: requiredText(address.ward_code, "Address.ward_code"),
      name: requiredText(address.ward_name, "Address.ward_name")
    },
    addressLine: requiredText(address.street_address, "Address.street_address"),
    deliveryNote: null,
    isDefault: address.is_default === true,
    version: positiveVersion(address.version, etag),
    updatedAt: requiredText(address.updated_at ?? address.created_at, "Address.updated_at")
  };
}

export function mapMvpAddressList(addresses: ShippingAddress[]): AddressList {
  const items = addresses.filter((address) => address.is_deleted !== true).map((address) => mapMvpAddress(address));
  return { items, total: items.length };
}

function addressLabelValue(type: AddressType): AddressWrite["label"] {
  return type === "GIFT" ? "GIFT_RECIPIENT" : type;
}

export function toMvpAddressWrite(input: AddressInput): AddressWrite {
  return {
    recipient_name: input.recipientName.trim(),
    phone_number: input.recipientPhone.trim(),
    street_address: input.addressLine.trim(),
    ward_code: input.ward.code,
    province_code: input.province.code,
    ward_name: input.ward.name,
    province_name: input.province.name,
    label: addressLabelValue(input.type),
    is_default: Boolean(input.isDefault)
  };
}

export function toMvpAddressUpdate(input: AddressInput): AddressUpdate {
  const write = toMvpAddressWrite(input);
  return {
    recipient_name: write.recipient_name,
    phone_number: write.phone_number,
    street_address: write.street_address,
    ward_code: write.ward_code,
    province_code: write.province_code,
    ward_name: write.ward_name,
    province_name: write.province_name,
    latitude: write.latitude,
    longitude: write.longitude,
    label: write.label
  };
}

export function fieldErrorsFromDetails(details: Record<string, unknown> | undefined, scope: "profile" | "address"): Record<string, string> {
  const errors = details?.errors;
  if (!Array.isArray(errors)) return {};
  const fieldNames: Record<string, string> = {
    full_name: "fullName",
    phone_number: scope === "profile" ? "phone" : "recipientPhone",
    recipient_name: "recipientName",
    street_address: "addressLine",
    province_code: "province",
    ward_code: "ward"
  };
  return Object.fromEntries(errors.flatMap((item) => {
    if (!item || typeof item !== "object") return [];
    const value = item as Record<string, unknown>;
    return typeof value.field === "string" && typeof value.message === "string"
      ? [[fieldNames[value.field] ?? value.field, value.message]]
      : [];
  }));
}
