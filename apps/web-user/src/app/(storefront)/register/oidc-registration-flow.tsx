"use client";

import { FormEvent, useRef, useState } from "react";
import Link from "next/link";
import { FormField, StatusNotice, SubmitButton } from "@/components/auth/form-controls";

export function OidcRegistrationFlow({ returnUrl }: { returnUrl: string }) {
  const key = useRef<string | null>(null);
  const [pending, setPending] = useState(false);
  const [done, setDone] = useState(false);
  const [message, setMessage] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true); setMessage("");
    const form = new FormData(event.currentTarget);
    key.current ??= crypto.randomUUID();
    try {
      const response = await fetch("/api/auth/register", { method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": key.current }, body: JSON.stringify(Object.fromEntries(form)) });
      const result = await response.json();
      if (!response.ok) throw new Error(result.message ?? "Chưa thể đăng ký.");
      if (result.status === "REGISTERED") setDone(true);
      else setMessage("Đăng ký đang được xử lý. Vui lòng chờ rồi gửi lại cùng yêu cầu để kiểm tra.");
    } catch (error) { setMessage(error instanceof Error ? error.message : "Chưa thể đăng ký."); }
    finally { setPending(false); }
  }
  if (done) return <div><StatusNotice tone="info">Đã tạo tài khoản. Hãy đăng nhập để kiểm tra hồ sơ. Hồ sơ mới cần hoàn tất xác minh trước khi mua hàng.</StatusNotice><Link className="primary-button" href={`/api/auth/oidc/login?returnUrl=${encodeURIComponent("/account/profile")}`}>Tiếp tục đăng nhập</Link></div>;
  return <form className="auth-form" onSubmit={submit}>
    {message && <StatusNotice>{message}</StatusNotice>}
    <FormField id="register-first-name" name="firstName" label="Tên" icon="♙" autoComplete="given-name" required />
    <FormField id="register-last-name" name="lastName" label="Họ và tên đệm" icon="♙" autoComplete="family-name" required />
    <FormField id="register-email" name="email" label="Email đăng nhập" icon="✉" type="email" autoComplete="email" required />
    <FormField id="register-password" name="password" label="Mật khẩu (12–128 ký tự)" icon="⌑" type="password" autoComplete="new-password" minLength={12} maxLength={128} required />
    <SubmitButton pending={pending}>Đăng ký tài khoản</SubmitButton>
    <p className="switch-flow"><Link href={`/login?returnUrl=${encodeURIComponent(returnUrl)}`}>Đã có tài khoản? Đăng nhập</Link></p>
  </form>;
}
