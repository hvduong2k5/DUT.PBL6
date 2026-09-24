"use client";

import Link from "next/link";
import { FormEvent, useEffect, useRef, useState } from "react";
import { FormField, PasswordChecklist, StatusNotice, SubmitButton } from "@/components/auth/form-controls";
import { RegistrationResult } from "@/lib/auth/types";
import { getFieldErrors, getSafeErrorMessage } from "@/lib/auth/ui-error";
import { isPasswordValid, isValidEmail } from "@/lib/auth/validation";
import { authService } from "@/services/auth-service";

export function RegistrationFlow({ verificationToken, returnUrl }: { verificationToken?: string; returnUrl: string }) {
  if (verificationToken) return <RegistrationVerification token={verificationToken} returnUrl={returnUrl} />;
  return <RegistrationForm returnUrl={returnUrl} />;
}

function RegistrationVerification({ token, returnUrl }: { token: string; returnUrl: string }) {
  const started = useRef(false);
  const [error, setError] = useState("");
  useEffect(() => {
    if (started.current) return;
    started.current = true;
    authService.confirmRegistration(token).then(() => window.location.assign(returnUrl)).catch((caught) => setError(getSafeErrorMessage(caught)));
  }, [returnUrl, token]);
  return error ? <><StatusNotice>{error}</StatusNotice><Link className="secondary-link full-width" href="/register">Bắt đầu đăng ký lại</Link></> : <div className="center-state"><span className="large-spinner" /><h2>Đang xác minh email...</h2><p>Vui lòng giữ nguyên cửa sổ này.</p></div>;
}

function RegistrationForm({ returnUrl }: { returnUrl: string }) {
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [result, setResult] = useState<RegistrationResult>();
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [resendMessage, setResendMessage] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const displayName = String(form.get("displayName") ?? "").trim();
    const email = String(form.get("email") ?? "").trim();
    const confirmation = String(form.get("passwordConfirmation") ?? "");
    const localErrors: Record<string, string> = {};
    if (!displayName) localErrors.displayName = "Vui lòng nhập tên hiển thị.";
    if (!isValidEmail(email)) localErrors.email = "Vui lòng nhập email hợp lệ.";
    if (!isPasswordValid(password)) localErrors.password = "Mật khẩu chưa đáp ứng đầy đủ điều kiện.";
    if (confirmation !== password) localErrors.passwordConfirmation = "Mật khẩu xác nhận chưa trùng khớp.";
    if (form.get("terms") !== "on") localErrors.termsVersion = "Bạn cần chấp nhận điều khoản.";
    if (Object.keys(localErrors).length) return setFieldErrors(localErrors);

    setPending(true); setError(""); setFieldErrors({});
    try {
      setResult(await authService.register({ displayName, email, password, termsVersion: process.env.NEXT_PUBLIC_TERMS_VERSION ?? "2026-09" }));
      setPassword("");
    } catch (caught) { setFieldErrors(getFieldErrors(caught)); setError(getSafeErrorMessage(caught)); }
    finally { setPending(false); }
  }

  async function resend() {
    if (!result) return;
    setPending(true); setError("");
    try { await authService.resendRegistration(result.verificationId); setResendMessage("Email xác minh mới đã được chấp nhận để gửi."); }
    catch (caught) { setError(getSafeErrorMessage(caught)); }
    finally { setPending(false); }
  }

  if (result) return (
    <div className="center-state email-sent"><span className="state-icon">✉</span><span className="eyebrow">Đăng ký thành công</span><h2>Kiểm tra hộp thư của bạn</h2><p>Chúng tôi đã gửi liên kết xác minh đến <strong>{result.delivery.maskedDestination}</strong>. Liên kết có hiệu lực trong 24 giờ.</p>{resendMessage ? <StatusNotice tone="success">{resendMessage}</StatusNotice> : null}{error ? <StatusNotice>{error}</StatusNotice> : null}<button className="secondary-button" onClick={resend} disabled={pending}>Gửi lại email xác minh</button><p className="dev-hint">Local mock: mở <code>/register?verificationToken=mock-token</code> để thử bước xác minh.</p></div>
  );

  const query = returnUrl === "/" ? "" : `?returnUrl=${encodeURIComponent(returnUrl)}`;
  return (
    <form className="auth-form registration-form" onSubmit={submit} noValidate>
      {error ? <StatusNotice>{error}</StatusNotice> : null}
      <div className="form-grid"><FormField id="displayName" name="displayName" label="Tên hiển thị" icon="♙" autoComplete="name" placeholder="Nguyễn Hoàng Bảo Anh" required error={fieldErrors.displayName} /><FormField id="register-email" name="email" type="email" label="Email" icon="✉" autoComplete="email" placeholder="quykhach@example.com" required error={fieldErrors.email} /></div>
      <FormField id="register-password" name="password" type="password" label="Mật khẩu khởi tạo" icon="⌑" autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} required error={fieldErrors.password} />
      <PasswordChecklist password={password} />
      <FormField id="password-confirmation" name="passwordConfirmation" type="password" label="Xác nhận mật khẩu" icon="⌑" autoComplete="new-password" required error={fieldErrors.passwordConfirmation} />
      <label className="checkbox terms"><input type="checkbox" name="terms" aria-describedby={fieldErrors.termsVersion ? "terms-error" : undefined} /><span>Tôi đồng ý với <a href="#terms">Quy chế Thẻ Tri Kỷ</a> và Chính sách Bảo mật.</span></label>
      {fieldErrors.termsVersion ? <span className="field-error" id="terms-error">{fieldErrors.termsVersion}</span> : null}
      <SubmitButton pending={pending}>Đăng Ký &amp; Gửi Email Xác Minh</SubmitButton>
      <p className="switch-flow">Đã sở hữu tài khoản? <Link href={`/login${query}`}>Đăng nhập ngay →</Link></p>
    </form>
  );
}
