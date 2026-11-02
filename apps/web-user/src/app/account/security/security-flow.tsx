"use client";

import { FormEvent, useState } from "react";
import { AccountShell } from "@/components/account/account-shell";
import { PasswordChecklist } from "@/components/auth/form-controls";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { useAuth } from "@/context/AuthContext";
import { getSafeErrorMessage } from "@/lib/auth/ui-error";
import { isPasswordValid } from "@/lib/auth/validation";
import { changePassword } from "@/services/auth-service";

export function SecurityFlow() {
  const { user } = useAuth();
  const [newPassword, setNewPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<{ kind: "success" | "error"; text: string } | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const currentPassword = String(form.get("currentPassword") ?? "");
    const confirmation = String(form.get("passwordConfirmation") ?? "");
    const localErrors: Record<string, string> = {};
    if (!currentPassword) localErrors.currentPassword = "Vui lòng nhập mật khẩu hiện tại.";
    if (!isPasswordValid(newPassword)) localErrors.newPassword = "Mật khẩu mới chưa đáp ứng đầy đủ điều kiện.";
    if (confirmation !== newPassword) localErrors.passwordConfirmation = "Mật khẩu xác nhận chưa trùng khớp.";
    if (currentPassword && currentPassword === newPassword) localErrors.newPassword = "Mật khẩu mới phải khác mật khẩu hiện tại.";
    setErrors(localErrors);
    if (Object.keys(localErrors).length) return;

    setPending(true);
    setNotice(null);
    try {
      await changePassword(currentPassword, newPassword);
      formElement.reset();
      setNewPassword("");
      setNotice({ kind: "success", text: "Mật khẩu đã được cập nhật thành công." });
    } catch (error) {
      setNotice({ kind: "error", text: getSafeErrorMessage(error) });
    } finally {
      setPending(false);
    }
  }

  const displayName = user?.displayName || [user?.firstName, user?.lastName].filter(Boolean).join(" ") || user?.email;
  return <><SiteHeader /><AccountShell active="security" customerName={displayName} customerEmail={user?.email}>
    <section className="account-hero"><div><span className="eyebrow">Bảo mật tài khoản</span><h1>Đổi mật khẩu</h1><p>Cập nhật mật khẩu dùng cho phương thức đăng nhập Email/Mật khẩu.</p></div></section>
    <form className="account-card profile-form security-form" onSubmit={submit} noValidate>
      <header className="card-heading"><span className="heading-icon">⌑</span><div><h2>Mật khẩu đăng nhập</h2><p>Mật khẩu được gửi trực tiếp đến Identity qua kết nối có cookie phiên HttpOnly.</p></div></header>
      {notice ? <div className={`status-notice ${notice.kind}`} role="status">{notice.text}</div> : null}
      <div className="profile-grid">
        <label className="account-field wide"><span>Mật khẩu hiện tại <b>*</b></span><input name="currentPassword" type="password" autoComplete="current-password" aria-invalid={Boolean(errors.currentPassword)} />{errors.currentPassword ? <small>{errors.currentPassword}</small> : null}</label>
        <label className="account-field wide"><span>Mật khẩu mới <b>*</b></span><input name="newPassword" type="password" autoComplete="new-password" value={newPassword} onChange={(event) => { setNewPassword(event.target.value); setErrors((current) => ({ ...current, newPassword: "" })); }} aria-invalid={Boolean(errors.newPassword)} />{errors.newPassword ? <small>{errors.newPassword}</small> : null}</label>
        <div className="wide"><PasswordChecklist password={newPassword} /></div>
        <label className="account-field wide"><span>Xác nhận mật khẩu mới <b>*</b></span><input name="passwordConfirmation" type="password" autoComplete="new-password" aria-invalid={Boolean(errors.passwordConfirmation)} />{errors.passwordConfirmation ? <small>{errors.passwordConfirmation}</small> : null}</label>
      </div>
      <footer className="form-footer"><span>Không chia sẻ mật khẩu hoặc mã phiên với người khác.</span><button className="primary-button" disabled={pending}>{pending ? "Đang cập nhật..." : "Cập nhật mật khẩu"}</button></footer>
    </form>
  </AccountShell><SiteFooter /></>;
}
