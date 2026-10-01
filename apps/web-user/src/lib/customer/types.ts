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

export interface ApiFieldError {
  field: string;
  code: string;
  message: string;
}

interface ApiErrorBody {
  code?: string;
  message?: string;
  error_code?: string;
  user_message?: string;
  errors?: ApiFieldError[];
  requestId?: string;
}

export class CustomerApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly errors: ApiFieldError[];
  readonly requestId?: string;

  constructor(status: number, body: ApiErrorBody) {
    super(body.user_message || body.message || "Không thể xử lý yêu cầu lúc này.");
    this.name = "CustomerApiError";
    this.status = status;
    this.code = body.error_code || body.code || "UNKNOWN_ERROR";
    this.errors = body.errors ?? [];
    this.requestId = body.requestId;
  }
}
