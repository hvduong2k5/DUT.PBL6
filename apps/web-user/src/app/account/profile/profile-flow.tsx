"use client";

import Link from "next/link";
import { FormEvent, useEffect, useRef, useState } from "react";
import { AccountShell } from "@/components/account/account-shell";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { CustomerApiError, CustomerProfile, ProfileInput } from "@/lib/customer/types";
import { FieldErrors, normalizePhone, validateProfile } from "@/lib/customer/validation";
import { customerService } from "@/services/customer-service";

const EMPTY: ProfileInput = { fullName: "", email: "", phone: null, dateOfBirth: null, gender: null };

function toForm(profile: CustomerProfile): ProfileInput {
  return { fullName: profile.fullName, email: profile.email, phone: profile.phone, dateOfBirth: profile.dateOfBirth, gender: profile.gender };
}

function formatUpdatedAt(value: string): string {
  return new Intl.DateTimeFormat("vi-VN", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

export function ProfileFlow() {
  const started = useRef(false);
  const [profile, setProfile] = useState<CustomerProfile | null>(null);
  const [form, setForm] = useState<ProfileInput>(EMPTY);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [notice, setNotice] = useState<{ kind: "success" | "error"; text: string } | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  async function loadProfile() {
    setLoading(true);
    setNotice(null);
    try {
      const result = await customerService.getProfile();
      setProfile(result);
      setForm(toForm(result));
    } catch (error) {
      setNotice({ kind: "error", text: error instanceof Error ? error.message : "Không thể tải hồ sơ." });
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    customerService.getProfile()
      .then((result) => { setProfile(result); setForm(toForm(result)); })
      .catch((error: unknown) => setNotice({ kind: "error", text: error instanceof Error ? error.message : "Không thể tải hồ sơ." }))
      .finally(() => setLoading(false));
  }, []);

  function update<K extends keyof ProfileInput>(key: K, value: ProfileInput[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setErrors((current) => ({ ...current, [key]: "" }));
    setNotice(null);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!profile) return;
    const clientErrors = validateProfile(form);
    setErrors(clientErrors);
    if (Object.keys(clientErrors).length) return;

    setSaving(true);
    setNotice(null);
    const payload: ProfileInput = {
      fullName: form.fullName.trim(),
      email: form.email.trim(),
      phone: form.phone ? normalizePhone(form.phone) : null,
      dateOfBirth: form.dateOfBirth || null,
      gender: form.gender
    };
    try {
      const result = await customerService.updateProfile(payload, profile.version);
      const merged = { ...result, ...payload };
      setProfile(merged);
      setForm(toForm(merged));
      setNotice({ kind: "success", text: "Hồ sơ Tri Kỷ đã được cập nhật." });
    } catch (error) {
      if (error instanceof CustomerApiError) {
        setErrors(Object.fromEntries(error.errors.map((item) => [item.field, item.message])));
        setNotice({ kind: "error", text: error.code === "VERSION_CONFLICT" ? "Hồ sơ vừa thay đổi ở nơi khác. Hãy tải lại dữ liệu mới nhất." : error.message });
      } else setNotice({ kind: "error", text: "Không thể lưu hồ sơ lúc này." });
    } finally {
      setSaving(false);
    }
  }

  return <><SiteHeader /><AccountShell active="profile" customerName={profile?.fullName} customerEmail={profile?.email}>
    <section className="account-hero">
      <div><span className="eyebrow">Quản trị Tri Kỷ</span><h1>Hồ sơ cá nhân</h1><p>Cập nhật thông tin liên hệ để hành trình đặt quà thuận tiện và chính xác.</p></div>
      {profile?.email ? <span className="verified-badge">Email liên hệ đã lưu</span> : null}
    </section>

    {loading ? <section className="account-card center-state" aria-live="polite"><span className="large-spinner" /><p>Đang mở hồ sơ Tri Kỷ…</p></section> : !profile ?
      <section className="account-card center-state"><span className="state-icon">!</span><h2>Chưa thể mở hồ sơ</h2><p>{notice?.text}</p><div className="state-actions"><button className="secondary-button" onClick={() => void loadProfile()}>Thử lại</button><Link className="primary-link" href="/login?returnUrl=/account/profile">Đăng nhập lại</Link></div></section> :
      <form className="account-card profile-form" onSubmit={submit} noValidate>
        <header className="card-heading"><span className="heading-icon">♙</span><div><h2>Thông tin định danh cơ bản</h2><p>Thông tin phục vụ định danh tài khoản và gợi ý cho Checkout.</p></div></header>
        {notice && <div className={`status-notice ${notice.kind}`} role="status">{notice.text}</div>}
        <div className="profile-grid">
          <label className="account-field wide"><span>Họ và tên <b>*</b></span><input value={form.fullName} onChange={(event) => update("fullName", event.target.value)} aria-invalid={Boolean(errors.fullName)} />{errors.fullName && <small>{errors.fullName}</small>}</label>
          <label className="account-field"><span>Số điện thoại tài khoản</span><div className="readonly-input"><span>{profile.phone ?? "Chưa có"}</span></div><em>API hiện chưa hỗ trợ đổi số điện thoại từ hồ sơ.</em></label>
          <label className="account-field"><span>Email liên hệ <b>*</b></span><input type="email" autoComplete="email" value={form.email} onChange={(event) => update("email", event.target.value)} aria-invalid={Boolean(errors.email)} />{errors.email ? <small>{errors.email}</small> : <em>API cho phép cập nhật email nhưng chưa cung cấp trạng thái xác minh.</em>}</label>
        </div>
        <footer className="form-footer"><span>Dữ liệu được tải lúc: {formatUpdatedAt(profile.updatedAt)}</span><div><button type="button" className="secondary-button" onClick={() => { setForm(toForm(profile)); setErrors({}); setNotice(null); }}>Hủy thay đổi</button><button className="primary-button" disabled={saving}>{saving && <span className="spinner" />}{saving ? "Đang lưu…" : "Lưu thay đổi"}</button></div></footer>
      </form>}
  </AccountShell><SiteFooter /></>;
}
