"use client";

import Link from "next/link";
import { type FormEvent, useState } from "react";
import { useAuth } from "@/components/auth/auth-provider";
import { PasswordChecklist, StatusNotice, SubmitButton } from "@/components/auth/form-controls";
import { getSafeErrorMessage } from "@/lib/auth/ui-error";
import { isPasswordValid } from "@/lib/auth/validation";
import { identityService } from "@/services/identity-service";

export function ChangePasswordFlow() {
  const { isAuthenticated, isLoading, isEmployee, login } = useAuth();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [pending, setPending] = useState(false);
  const [notice, setNotice] = useState<{ tone: "success" | "error"; text: string }>();

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!currentPassword) return setNotice({ tone: "error", text: "Vui lòng nhập mật khẩu hiện tại." });
    if (!isPasswordValid(newPassword)) return setNotice({ tone: "error", text: "Mật khẩu mới chưa đáp ứng đầy đủ điều kiện." });
    if (newPassword !== confirmation) return setNotice({ tone: "error", text: "Mật khẩu xác nhận chưa trùng khớp." });
    setPending(true); setNotice(undefined);
    try {
      await identityService.changePassword(currentPassword, newPassword);
      setCurrentPassword(""); setNewPassword(""); setConfirmation("");
      setNotice({ tone: "success", text: "Mật khẩu đã được cập nhật thành công." });
    } catch (cause) {
      setNotice({ tone: "error", text: getSafeErrorMessage(cause) });
    } finally { setPending(false); }
  }

  if (isLoading) return <main className="session-state" aria-busy="true"><span className="large-spinner" /><h1>Đang kiểm tra phiên đăng nhập…</h1></main>;
  if (!isAuthenticated) return <main className="session-state"><span>⌑</span><h1>Đăng nhập để đổi mật khẩu</h1><button className="primary-button" type="button" onClick={() => login("/settings/password")}>Đăng nhập an toàn</button></main>;

  return <main className="account-page"><nav className="breadcrumbs"><Link href={isEmployee ? "/admin" : "/account/profile"}>{isEmployee ? "Kênh Quản Trị" : "Tài khoản"}</Link><span>›</span><strong>Đổi mật khẩu</strong></nav><section className="account-card profile-form"><header className="card-heading"><span className="heading-icon">⌑</span><div><h1>Đổi mật khẩu</h1><p>Mật khẩu được gửi trực tiếp đến Identity Service qua phiên cookie HttpOnly.</p></div></header>{notice ? <StatusNotice tone={notice.tone}>{notice.text}</StatusNotice> : null}<form className="auth-form" onSubmit={submit}><label className="account-field"><span>Mật khẩu hiện tại</span><input type="password" autoComplete="current-password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} required /></label><label className="account-field"><span>Mật khẩu mới</span><input type="password" autoComplete="new-password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} required /></label><PasswordChecklist password={newPassword} /><label className="account-field"><span>Xác nhận mật khẩu mới</span><input type="password" autoComplete="new-password" value={confirmation} onChange={(event) => setConfirmation(event.target.value)} required /></label><SubmitButton pending={pending}>Cập nhật mật khẩu</SubmitButton></form></section></main>;
}
