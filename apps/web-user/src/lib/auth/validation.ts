export const PASSWORD_RULES = [
  "PASSWORD_TOO_SHORT",
  "PASSWORD_TOO_LONG",
  "PASSWORD_MISSING_UPPERCASE",
  "PASSWORD_MISSING_LOWERCASE",
  "PASSWORD_MISSING_DIGIT",
  "PASSWORD_MISSING_SPECIAL"
] as const;

export type PasswordRuleCode = (typeof PASSWORD_RULES)[number];

export interface PasswordCheck {
  code: PasswordRuleCode;
  label: string;
  valid: boolean;
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/u;
const UPPERCASE_PATTERN = /\p{Lu}/u;
const LOWERCASE_PATTERN = /\p{Ll}/u;
const DIGIT_PATTERN = /\p{Nd}/u;
const SPECIAL_PATTERN = /[\p{P}\p{S}]/u;

export function isValidEmail(value: string): boolean {
  return EMAIL_PATTERN.test(value.trim());
}

export function passwordCodePointLength(value: string): number {
  return Array.from(value).length;
}

export function getPasswordChecks(value: string): PasswordCheck[] {
  const length = passwordCodePointLength(value);

  return [
    { code: "PASSWORD_TOO_SHORT", label: "Từ 15 ký tự", valid: length >= 15 },
    { code: "PASSWORD_TOO_LONG", label: "Không quá 128 ký tự", valid: length <= 128 },
    { code: "PASSWORD_MISSING_UPPERCASE", label: "Có chữ hoa", valid: UPPERCASE_PATTERN.test(value) },
    { code: "PASSWORD_MISSING_LOWERCASE", label: "Có chữ thường", valid: LOWERCASE_PATTERN.test(value) },
    { code: "PASSWORD_MISSING_DIGIT", label: "Có chữ số", valid: DIGIT_PATTERN.test(value) },
    {
      code: "PASSWORD_MISSING_SPECIAL",
      label: "Có ký tự đặc biệt (không tính khoảng trắng)",
      valid: SPECIAL_PATTERN.test(value)
    }
  ];
}

export function isPasswordValid(value: string): boolean {
  return getPasswordChecks(value).every((check) => check.valid);
}
