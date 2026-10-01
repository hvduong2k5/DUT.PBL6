"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { AccountShell } from "@/components/account/account-shell";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import type { LoyaltyPoints } from "@/lib/customer-core/types";
import { CustomerCoreApiError } from "@/lib/customer-core/types";
import { customerCoreService } from "@/services/customer-core-service";

const VND = new Intl.NumberFormat("vi-VN", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const TIER_LABEL: Record<LoyaltyPoints["tier"], string> = { BRONZE: "Đồng", SILVER: "Bạc", GOLD: "Vàng", DIAMOND: "Kim Cương" };

export function LoyaltyFlow() {
  const searchParams = useSearchParams();
  const scenario = searchParams.get("mockScenario") ?? undefined;
  const [points, setPoints] = useState<LoyaltyPoints>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<{ status?: number; message: string }>();
  const [retryKey, setRetryKey] = useState(0);

  useEffect(() => {
    let active = true;
    customerCoreService.getLoyaltyPoints(scenario)
      .then((payload) => { if (active) setPoints(payload); })
      .catch((cause: unknown) => {
        if (!active) return;
        setError(cause instanceof CustomerCoreApiError ? { status: cause.status, message: cause.message } : { message: "Chưa thể tải ví điểm." });
      })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [retryKey, scenario]);

  function retry() {
    setLoading(true);
    setError(undefined);
    setRetryKey((value) => value + 1);
  }

  return <><SiteHeader /><AccountShell active="loyalty">
    <section className="account-hero"><div><span className="eyebrow">Thành viên Tri Kỷ</span><h1>Ví điểm &amp; đặc quyền</h1><p>Theo dõi số điểm khả dụng, hạng thành viên và giá trị quy đổi hiện tại.</p></div><Link className="verified-badge" href="/promotions">Xem kho ưu đãi →</Link></section>
    {loading ? <section className="account-card center-state" aria-busy="true"><span className="large-spinner" /><p>Đang mở ví điểm Tri Kỷ…</p></section> : null}
    {!loading && error ? <section className="account-card center-state" role="alert"><span className="state-icon">!</span><h2>Chưa thể mở ví điểm</h2><p>{error.message}</p><div className="state-actions">{error.status !== 401 ? <button className="secondary-button" type="button" onClick={retry}>Thử lại</button> : null}<Link className="primary-link" href="/login?returnUrl=/account/loyalty">Đăng nhập</Link></div></section> : null}
    {!loading && !error && points ? <div className="loyalty-layout">
      <section className="loyalty-wallet"><div className="loyalty-wallet-top"><span>Ô MẠ · TRI KỶ</span><strong>Hạng {TIER_LABEL[points.tier]}</strong></div><div className="loyalty-balance"><small>Điểm khả dụng</small><strong>{points.current_points.toLocaleString("vi-VN")}</strong><span>Điểm Di Sản</span></div><div className="loyalty-wallet-bottom"><span>Mã thành viên</span><strong>{points.customer_id}</strong></div></section>
      <section className="account-card loyalty-value"><p className="eyebrow">Giá trị hiện tại</p><h2>{VND.format(points.current_points * points.points_to_vnd_rate)}</h2><p>Tạm tính theo tỷ lệ <strong>1 điểm = {VND.format(points.points_to_vnd_rate)}</strong>.</p><small>Khả năng sử dụng điểm và số tiền cuối cùng được xác nhận theo chính sách tại Checkout.</small><Link className="primary-link" href="/promotions">Khám phá ưu đãi</Link></section>
      <section className="account-card loyalty-rules"><header className="card-heading"><span className="heading-icon">♡</span><div><h2>Quyền lợi đang hiển thị</h2><p>Dữ liệu được cung cấp trực tiếp từ tài khoản thành viên.</p></div></header><div><article><span>01</span><div><strong>Tích lũy theo giao dịch hợp lệ</strong><p>Điểm được cập nhật khi nghiệp vụ mua hàng hoàn tất theo chính sách.</p></div></article><article><span>02</span><div><strong>Hạng thành viên rõ ràng</strong><p>Hạng hiện tại của bạn là {TIER_LABEL[points.tier]}; hệ thống không tự suy diễn ngưỡng nâng hạng chưa có trong API.</p></div></article><article><span>03</span><div><strong>Quy đổi minh bạch</strong><p>Mọi lần áp dụng điểm cần hiển thị rõ giá trị trước khi xác nhận đơn.</p></div></article></div></section>
    </div> : null}
  </AccountShell><SiteFooter /></>;
}
