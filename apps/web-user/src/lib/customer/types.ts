export type Gender = "MALE" | "FEMALE" | "OTHER" | "PREFER_NOT_TO_SAY";
export type AddressType = "HOME" | "OFFICE" | "GIFT" | "OTHER";

export interface AdministrativeArea {
  code: string;
  name: string;
}

export interface CustomerProfile {
  id: string;
  fullName: string;
  email: string;
  emailVerified: boolean;
  phone: string | null;
  dateOfBirth: string | null;
  gender: Gender | null;
  version: number;
  updatedAt: string;
}

export interface CustomerAddress {
  id: string;
  label: string;
  type: AddressType;
  recipientName: string;
  recipientPhone: string;
  province: AdministrativeArea;
  ward: AdministrativeArea;
  addressLine: string;
  deliveryNote: string | null;
  isDefault: boolean;
  version: number;
  updatedAt: string;
}

export interface AddressList {
  items: CustomerAddress[];
  total: number;
}

export interface ProfileInput {
  fullName: string;
  email: string;
  phone: string | null;
  dateOfBirth: string | null;
  gender: Gender | null;
}

export interface AddressInput {
  label: string;
  type: AddressType;
  recipientName: string;
  recipientPhone: string;
  province: AdministrativeArea;
  ward: AdministrativeArea;
  addressLine: string;
  deliveryNote: string | null;
  isDefault?: boolean;
}
