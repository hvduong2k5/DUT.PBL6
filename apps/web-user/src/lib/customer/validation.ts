import type { AddressInput, ProfileInput } from "./types";

export type FieldErrors = Record<string, string>;

const PHONE_ALLOWED = /^[+\d\s().-]+$/u;

export function normalizePhone(value: string): string {
  return value.trim().replace(/[\s().-]/gu, "");
}

export function isValidPhone(value: string): boolean {
  if (!PHONE_ALLOWED.test(value.trim())) return false;
  const normalized = normalizePhone(value);
  const digits = normalized.startsWith("+") ? normalized.slice(1) : normalized;
  return (/^\d{8,15}$/u.test(digits) && !digits.startsWith("0")) || /^0\d{8,10}$/u.test(digits);
}

export function validateProfile(input: ProfileInput): FieldErrors {
  const errors: FieldErrors = {};
  const fullName = input.fullName.trim();

  if (fullName.length < 2) errors.fullName = "Họ và tên phải có ít nhất 2 ký tự.";
  if (fullName.length > 100) errors.fullName = "Họ và tên không được quá 100 ký tự.";
  if (input.phone && !isValidPhone(input.phone)) errors.phone = "Số điện thoại chưa đúng định dạng.";
  if (input.dateOfBirth && input.dateOfBirth > new Date().toISOString().slice(0, 10)) {
    errors.dateOfBirth = "Ngày sinh không được ở tương lai.";
  }

  return errors;
}

export function validateAddress(input: AddressInput): FieldErrors {
  const errors: FieldErrors = {};
  if (!input.label.trim()) errors.label = "Vui lòng đặt tên cho địa chỉ.";
  else if (input.label.trim().length > 50) errors.label = "Tên địa chỉ không được quá 50 ký tự.";
  if (input.recipientName.trim().length < 2) errors.recipientName = "Vui lòng nhập đầy đủ tên người nhận.";
  if (!isValidPhone(input.recipientPhone)) errors.recipientPhone = "Số điện thoại chưa đúng định dạng.";
  if (!input.province.code) errors.province = "Vui lòng chọn tỉnh hoặc thành phố.";
  if (!input.ward.code) errors.ward = "Vui lòng chọn phường hoặc xã.";
  if (input.addressLine.trim().length < 5) errors.addressLine = "Vui lòng nhập địa chỉ chi tiết.";
  if (input.addressLine.trim().length > 200) errors.addressLine = "Địa chỉ chi tiết không được quá 200 ký tự.";
  if ((input.deliveryNote?.length ?? 0) > 500) errors.deliveryNote = "Ghi chú không được quá 500 ký tự.";
  return errors;
}
