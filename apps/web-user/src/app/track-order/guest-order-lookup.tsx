"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import type { GuestChallenge } from "@/lib/orders/types";
import { OrderApiError } from "@/lib/orders/types";
import { orderService } from "@/services/order-service";

export function GuestOrderLookup() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [orderNumber, setOrderNumber] = useState("");
  const [contact, setContact] = useState("");
  const [otp, setOtp] = useState("");
  const [challenge, setChallenge] = useState<GuestChallenge | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function requestCode(event: FormEvent) {
    event.preventDefault(); setError(""); setLoading(true);
    try { setChallenge(await orderService.createGuestChallenge({ orderNumber, contact }, scenario)); }
    catch (cause) { setError(cause instanceof OrderApiError ? cause.message : "Không thể gửi mã xác minh."); }
    finally { setLoading(false); }
  }

  async function verify(event: FormEvent) {
    event.preventDefault(); if (!challenge) return; setError(""); setLoading(true);
    try { const result = await orderService.verifyGuestChallenge(challenge.challengeId, otp, scenario); router.push(result.orderPath); }
    catch (cause) { setError(cause instanceof OrderApiError ? cause.message : "Không thể xác minh Guest Order."); }
    finally { setLoading(false); }
  }

  return <main className="guest-order-page"><nav className="detail-breadcrumbs"><Link href="/">Trang chủ</Link><span>›</span><strong>Tra cứu Order</strong></nav><section className="guest-order-hero"><span>⌕</span><p className="eyebrow">Dành cho khách vãng lai • Bảo mật</p><h1>Tra cứu hành trình Order</h1><p>Nhập thông tin đã dùng đặt hàng và xác minh mã một lần. Mã Order đơn lẻ không đủ quyền truy cập.</p></section><div className="guest-order-layout"><section className="guest-lookup-card"><header><span>▣</span><div><h2>Xác minh quyền sở hữu Order</h2><p>{challenge ? "Bước 2/2 — nhập mã xác minh" : "Bước 1/2 — xác nhận Order và thông tin liên hệ"}</p></div></header>{!challenge ? <form onSubmit={requestCode}><label><span>Mã Order <b>*</b></span><input value={orderNumber} onChange={(event) => setOrderNumber(event.target.value)} placeholder="OMA-260926-001" /></label><label><span>Số điện thoại hoặc email đặt hàng <b>*</b></span><input value={contact} onChange={(event) => setContact(event.target.value)} placeholder="0914 288 668" /></label><div className="guest-privacy-note">Thông tin được dùng để xác minh và không được hiển thị trước khi quyền sở hữu hợp lệ.</div>{error ? <div className="status-notice error" role="alert">{error}</div> : null}<button className="primary-button full-width" disabled={loading}>{loading ? "Đang gửi…" : "Gửi mã xác minh"}</button></form> : <form onSubmit={verify}><div className="status-notice info">{challenge.message} Điểm nhận: <strong>{challenge.maskedDestination}</strong></div><label><span>Mã xác minh 6 chữ số <b>*</b></span><input value={otp} onChange={(event) => setOtp(event.target.value.replace(/\D/gu, "").slice(0, 6))} inputMode="numeric" autoComplete="one-time-code" placeholder="789214" /></label><p className="guest-local-hint">Fixture local: OTP <strong>789214</strong>. OTP thật không được log hoặc đưa vào URL.</p>{error ? <div className="status-notice error" role="alert">{error}</div> : null}<button className="primary-button full-width" disabled={loading || otp.length !== 6}>{loading ? "Đang xác minh…" : "Xác minh & mở Order"}</button><button type="button" className="guest-reset-button" onClick={() => { setChallenge(null); setOtp(""); setError(""); }}>Dùng thông tin khác</button></form>}</section><aside className="guest-order-aside"><section><span>◈</span><h2>Order được bảo vệ thế nào?</h2><ul><li>Không thể tra cứu chỉ bằng mã Order.</li><li>OTP có thời hạn và giới hạn số lần thử.</li><li>Guest access chỉ mở đúng một Order.</li><li>Dữ liệu nhạy cảm được che khi cần thiết.</li></ul></section><section><h2>Cần trợ giúp nhanh?</h2><p>Hotline Tri Kỷ: <strong>1900 68 Hue</strong></p><Link href="/login?returnUrl=/account/orders">Đăng nhập tài khoản Tri Kỷ</Link></section></aside></div></main>;
}
