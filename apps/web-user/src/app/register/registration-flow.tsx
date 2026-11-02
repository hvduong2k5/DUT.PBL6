"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { FormField, PasswordChecklist, StatusNotice, SubmitButton } from "@/components/auth/form-controls";
import { useAuth } from "@/context/AuthContext";
import { getFieldErrors, getSafeErrorMessage } from "@/lib/auth/ui-error";
import { isPasswordValid, isValidEmail } from "@/lib/auth/validation";
import { registerCustomer } from "@/services/auth-service";

export function RegistrationFlow({ returnUrl }: { returnUrl: string }) {
  const { login } = useAuth();
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const firstName = String(form.get("firstName") ?? "").trim();
    const lastName = String(form.get("lastName") ?? "").trim();
    const email = String(form.get("email") ?? "").trim();
    const confirmation = String(form.get("passwordConfirmation") ?? "");
    const localErrors: Record<string, string> = {};
    if (!firstName) localErrors.firstName = "Vui lòng nhập họ.";
    if (!lastName) localErrors.lastName = "Vui lòng nhập tên.";
    if (!isValidEmail(email)) localErrors.email = "Email chưa hợp lệ.";
    if (!isPasswordValid(password)) localErrors.rawPassword = "Mật khẩu chưa đáp ứng đầy đủ điều kiện.";
    if (confirmation !== password) localErrors.passwordConfirmation = "Mật khẩu xác nhận chưa trùng khớp.";
    if (form.get("terms") !== "on") localErrors.terms = "Bạn cần chấp nhận điều khoản.";
    if (Object.keys(localErrors).length) return setFieldErrors(localErrors);

    setPending(true);
    setError("");
    setFieldErrors({});
    try {
      await registerCustomer({ firstName, lastName, email, rawPassword: password });
      setSuccess(true);
      window.setTimeout(() => login(returnUrl), 1200);
    } catch (caught) {
      setFieldErrors(getFieldErrors(caught));
      setError(getSafeErrorMessage(caught));
      setPending(false);
    }
  }

  const query = returnUrl === "/" ? "" : `?returnUrl=${encodeURIComponent(returnUrl)}`;
  return (
    <form className="auth-form registration-form" onSubmit={submit} noValidate>
      {success ? <div className="auth-toast" role="status">Đăng ký thành công! Đang chuyển đến màn hình đăng nhập...</div> : null}
      {error ? <StatusNotice>{error}</StatusNotice> : null}
      <div className="form-grid">
        <FormField id="first-name" name="firstName" label="Họ" icon="♙" autoComplete="family-name" placeholder="Nguyễn" required error={fieldErrors.firstName} />
        <FormField id="last-name" name="lastName" label="Tên" icon="♙" autoComplete="given-name" placeholder="Văn An" required error={fieldErrors.lastName} />
      </div>
      <FormField id="register-email" name="email" type="email" label="Email" icon="✉" autoComplete="email" placeholder="quykhach@example.com" required error={fieldErrors.email} />
      <FormField id="register-password" name="rawPassword" type="password" label="Mật khẩu" icon="⌑" autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} required error={fieldErrors.rawPassword} />
      <PasswordChecklist password={password} />
      <FormField id="password-confirmation" name="passwordConfirmation" type="password" label="Xác nhận mật khẩu" icon="⌑" autoComplete="new-password" required error={fieldErrors.passwordConfirmation} />
      <label className="checkbox terms"><input type="checkbox" name="terms" aria-describedby={fieldErrors.terms ? "terms-error" : undefined} /><span>Tôi đồng ý với <a href="#terms">Quy chế Thẻ Tri Kỷ</a> và Chính sách Bảo mật.</span></label>
      {fieldErrors.terms ? <span className="field-error" id="terms-error">{fieldErrors.terms}</span> : null}
      <SubmitButton pending={pending}>{success ? "Đang chuyển hướng..." : "Đăng Ký Tài Khoản"}</SubmitButton>
      <button className="secondary-button full-width google-register-button" type="button" onClick={() => login(returnUrl)} disabled={pending}>Đăng ký nhanh bằng Google</button>
      <p className="switch-flow">Đã sở hữu tài khoản? <Link href={`/login${query}`}>Đăng nhập ngay →</Link></p>
    </form>
  );
}
