"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { FormField, PasswordChecklist, StatusNotice, SubmitButton } from "@/components/auth/form-controls";
import { getFieldErrors, getSafeErrorMessage } from "@/lib/auth/ui-error";
import { isPasswordValid, isValidEmail, isValidPhoneNumber } from "@/lib/auth/validation";
import { authService } from "@/services/auth-service";

export function RegistrationFlow({ verificationToken, returnUrl }: { verificationToken?: string; returnUrl: string }) {
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const fullName = String(form.get("fullName") ?? "").trim();
    const phoneNumber = String(form.get("phoneNumber") ?? "").trim();
    const email = String(form.get("email") ?? "").trim();
    const confirmation = String(form.get("passwordConfirmation") ?? "");
    const localErrors: Record<string, string> = {};
    if (!fullName) localErrors.fullName = "Vui lòng nhập họ và tên.";
    if (!isValidPhoneNumber(phoneNumber)) localErrors.phoneNumber = "Vui lòng nhập số điện thoại Việt Nam hợp lệ.";
    if (email && !isValidEmail(email)) localErrors.email = "Email chưa hợp lệ.";
    if (!isPasswordValid(password)) localErrors.password = "Mật khẩu chưa đáp ứng đầy đủ điều kiện.";
    if (confirmation !== password) localErrors.passwordConfirmation = "Mật khẩu xác nhận chưa trùng khớp.";
    if (form.get("terms") !== "on") localErrors.terms = "Bạn cần chấp nhận điều khoản.";
    if (Object.keys(localErrors).length) return setFieldErrors(localErrors);

    setPending(true); setError(""); setFieldErrors({});
    try {
      await authService.register({ fullName, phoneNumber, email: email || undefined, password });
      window.location.assign(returnUrl);
    } catch (caught) {
      setFieldErrors(getFieldErrors(caught));
      setError(getSafeErrorMessage(caught));
    } finally { setPending(false); }
  }

  const query = returnUrl === "/" ? "" : `?returnUrl=${encodeURIComponent(returnUrl)}`;
  return (
    <form className="auth-form registration-form" onSubmit={submit} noValidate>
      {verificationToken ? <StatusNotice tone="info">API customer không dùng bước xác minh email; bạn có thể đăng ký trực tiếp bằng số điện thoại.</StatusNotice> : null}
      {error ? <StatusNotice>{error}</StatusNotice> : null}
      <div className="form-grid"><FormField id="full-name" name="fullName" label="Họ và tên" icon="♙" autoComplete="name" placeholder="Nguyễn Văn An" required error={fieldErrors.fullName} /><FormField id="register-phone" name="phoneNumber" type="tel" label="Số điện thoại" icon="☎" autoComplete="tel" placeholder="0905 123 456" required error={fieldErrors.phoneNumber} /></div>
      <FormField id="register-email" name="email" type="email" label="Email (không bắt buộc)" icon="✉" autoComplete="email" placeholder="quykhach@example.com" error={fieldErrors.email} />
      <FormField id="register-password" name="password" type="password" label="Mật khẩu khởi tạo" icon="⌑" autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} required error={fieldErrors.password} />
      <PasswordChecklist password={password} />
      <FormField id="password-confirmation" name="passwordConfirmation" type="password" label="Xác nhận mật khẩu" icon="⌑" autoComplete="new-password" required error={fieldErrors.passwordConfirmation} />
      <label className="checkbox terms"><input type="checkbox" name="terms" aria-describedby={fieldErrors.terms ? "terms-error" : undefined} /><span>Tôi đồng ý với <a href="#terms">Quy chế Thẻ Tri Kỷ</a> và Chính sách Bảo mật.</span></label>
      {fieldErrors.terms ? <span className="field-error" id="terms-error">{fieldErrors.terms}</span> : null}
      <SubmitButton pending={pending}>Đăng Ký Tài Khoản</SubmitButton>
      <p className="switch-flow">Đã sở hữu tài khoản? <Link href={`/login${query}`}>Đăng nhập ngay →</Link></p>
    </form>
  );
}
