"use client";

import Link from "next/link";
import { FormEvent, useEffect, useRef, useState } from "react";
import { FormField, PasswordChecklist, StatusNotice, SubmitButton } from "@/components/auth/form-controls";
import { getFieldErrors, getSafeErrorMessage } from "@/lib/auth/ui-error";
import { isPasswordValid, isValidEmail } from "@/lib/auth/validation";
import { authService } from "@/services/auth-service";

type Phase = "request" | "sent" | "verifying" | "reset" | "success";

export function RecoveryFlow({ recoveryToken, returnUrl }: { recoveryToken?: string; returnUrl: string }) {
  const [phase, setPhase] = useState<Phase>(recoveryToken ? "verifying" : "request");
  const [email, setEmail] = useState("");
  const [resetProof, setResetProof] = useState("");
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const started = useRef(false);

  useEffect(() => {
    if (!recoveryToken || started.current) return;
    started.current = true;
    authService.verifyRecovery(recoveryToken).then((result) => { setResetProof(result.resetProof); setPhase("reset"); }).catch((caught) => { setError(getSafeErrorMessage(caught)); setPhase("request"); });
  }, [recoveryToken]);

  async function requestRecovery(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const value = String(form.get("email") ?? "").trim();
    if (!isValidEmail(value)) return setFieldErrors({ email: "Vui lòng nhập email hợp lệ." });
    setPending(true); setError(""); setFieldErrors({});
    try { await authService.requestRecovery(value); setEmail(value); setPhase("sent"); }
    catch (caught) { setError(getSafeErrorMessage(caught)); setFieldErrors(getFieldErrors(caught)); }
    finally { setPending(false); }
  }

  async function reset(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const confirmation = String(form.get("newPasswordConfirmation") ?? "");
    const localErrors: Record<string, string> = {};
    if (!isPasswordValid(password)) localErrors.newPassword = "Mật khẩu chưa đáp ứng đầy đủ điều kiện.";
    if (confirmation !== password) localErrors.newPasswordConfirmation = "Mật khẩu xác nhận chưa trùng khớp.";
    if (Object.keys(localErrors).length) return setFieldErrors(localErrors);
    setPending(true); setError(""); setFieldErrors({});
    try { await authService.resetPassword(resetProof, password); setPassword(""); setResetProof(""); setPhase("success"); }
    catch (caught) { setError(getSafeErrorMessage(caught)); setFieldErrors(getFieldErrors(caught)); }
    finally { setPending(false); }
  }

  const loginHref = returnUrl === "/" ? "/dang-nhap" : `/dang-nhap?returnUrl=${encodeURIComponent(returnUrl)}`;

  if (phase === "verifying") return <div className="center-state"><span className="large-spinner" /><h2>Đang kiểm tra liên kết...</h2></div>;
  if (phase === "sent") return <div className="center-state email-sent"><span className="state-icon">✉</span><h2>Hãy kiểm tra email</h2><p>Nếu <strong>{email}</strong> phù hợp với tài khoản, hướng dẫn khôi phục sẽ được gửi đến địa chỉ đó.</p><StatusNotice tone="info">Vì lý do riêng tư, thông báo này không xác nhận tài khoản có tồn tại hay không.</StatusNotice><button className="secondary-button" onClick={() => setPhase("request")}>Gửi một yêu cầu khác</button><p className="dev-hint">Local mock: mở <code>/khoi-phuc-quyen-truy-cap?recoveryToken=mock-token</code>.</p></div>;
  if (phase === "success") return <div className="center-state"><span className="state-icon success">✓</span><h2>Mật khẩu đã được cập nhật</h2><p>Tất cả phiên cũ đã được thu hồi. Hãy đăng nhập lại bằng mật khẩu mới.</p><Link className="primary-link full-width" href={loginHref}>Quay lại đăng nhập</Link></div>;
  if (phase === "reset") return <form className="auth-form" onSubmit={reset} noValidate>{error ? <StatusNotice>{error}</StatusNotice> : null}<StatusNotice tone="success">Liên kết hợp lệ. Hãy tạo mật khẩu mới cho tài khoản.</StatusNotice><FormField id="new-password" name="newPassword" type="password" label="Mật khẩu mới" icon="⌑" autoComplete="new-password" value={password} onChange={(event) => setPassword(event.target.value)} required error={fieldErrors.newPassword} /><PasswordChecklist password={password} /><FormField id="new-password-confirmation" name="newPasswordConfirmation" type="password" label="Xác nhận mật khẩu mới" icon="⌑" autoComplete="new-password" required error={fieldErrors.newPasswordConfirmation} /><SubmitButton pending={pending}>Xác Nhận &amp; Đặt Lại Mật Khẩu</SubmitButton></form>;

  return <form className="auth-form" onSubmit={requestRecovery} noValidate>{error ? <StatusNotice>{error}</StatusNotice> : null}<FormField id="recovery-email" name="email" type="email" label="Email đã đăng ký" icon="✉" autoComplete="email" placeholder="tri.ky@example.com" required error={fieldErrors.email} hint="Chúng tôi sẽ luôn trả về cùng một thông báo để bảo vệ tài khoản." /><SubmitButton pending={pending}>Gửi Liên Kết Khôi Phục</SubmitButton><Link className="back-link" href={loginHref}>← Quay lại Đăng nhập Tri Kỷ</Link></form>;
}
