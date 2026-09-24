"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { FormField, StatusNotice, SubmitButton } from "@/components/auth/form-controls";
import { isValidEmail } from "@/lib/auth/validation";
import { getFieldErrors, getSafeErrorMessage } from "@/lib/auth/ui-error";
import { authService } from "@/services/auth-service";

export function LoginForm({ returnUrl }: { returnUrl: string }) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const email = String(form.get("email") ?? "").trim();
    const password = String(form.get("password") ?? "");
    const localErrors: Record<string, string> = {};
    if (!isValidEmail(email)) localErrors.email = "Vui lòng nhập email hợp lệ.";
    if (!password) localErrors.password = "Vui lòng nhập mật khẩu.";
    if (Object.keys(localErrors).length) return setFieldErrors(localErrors);

    setPending(true); setError(""); setFieldErrors({});
    try {
      await authService.login({ email, password, rememberMe: form.get("rememberMe") === "on" });
      window.location.assign(returnUrl);
    } catch (caught) {
      setFieldErrors(getFieldErrors(caught));
      setError(getSafeErrorMessage(caught));
    } finally { setPending(false); }
  }

  const query = returnUrl === "/" ? "" : `?returnUrl=${encodeURIComponent(returnUrl)}`;

  return (
    <form className="auth-form" onSubmit={submit} noValidate>
      {error ? <StatusNotice>{error}</StatusNotice> : null}
      <FormField id="email" name="email" type="email" label="Email Tri Kỷ" icon="✉" placeholder="tri.ky@example.com" autoComplete="email" required error={fieldErrors.email} />
      <FormField id="password" name="password" type="password" label="Mật khẩu" icon="⌑" placeholder="Nhập mật khẩu của bạn" autoComplete="current-password" required error={fieldErrors.password} />
      <div className="form-options"><label className="checkbox"><input type="checkbox" name="rememberMe" /> <span>Ghi nhớ đăng nhập trên thiết bị này</span></label><Link href={`/khoi-phuc-quyen-truy-cap${query}`}>Quên mật khẩu?</Link></div>
      <SubmitButton pending={pending}>Hồi Cung Đăng Nhập <span aria-hidden="true">→</span></SubmitButton>
      <p className="switch-flow">Chưa có tài khoản Tri Kỷ? <Link href={`/dang-ky${query}`}>Đăng ký ngay</Link></p>
      {returnUrl.startsWith("/thanh-toan") ? <Link className="guest-link" href={returnUrl}>Tiếp tục thanh toán với tư cách khách</Link> : null}
      <div className="security-note">♢ Phiên đăng nhập được bảo vệ bằng cookie HttpOnly; mật khẩu không được lưu trên trình duyệt.</div>
    </form>
  );
}
