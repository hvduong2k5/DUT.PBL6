"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { useAuth } from "./auth-provider";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";

export function CustomerPurchaseGuard({ children }: { children: ReactNode }) {
  const { status, isEmployee, isLoading } = useAuth();

  if (isLoading) {
    return <main className="session-state" aria-busy="true"><span className="large-spinner" /><h1>Đang kiểm tra quyền mua hàng…</h1></main>;
  }
  if (isEmployee) {
    return <main className="session-state"><span>!</span><h1>Tài khoản nhân viên không thể mua hàng</h1><p>Hãy đăng xuất và sử dụng tài khoản Customer cá nhân để tạo giỏ hàng, Checkout, đơn hàng hoặc địa chỉ nhận hàng.</p><div className="state-actions"><Link className="primary-link" href="/admin">Đến Kênh Quản Trị</Link><Link className="secondary-button" href="/">Về trang chủ</Link></div></main>;
  }
  if (status === "guest" && !MVP_FEATURE_GATES.guestOrder) return <main className="session-state"><h1>Đăng nhập để mua hàng</h1><p>Vui lòng đăng nhập để dùng giỏ hàng và thanh toán.</p><Link className="primary-link" href="/login?returnUrl=%2Fcart">Đăng nhập</Link></main>;
  if (status === "error") return <main className="session-state"><h1>Chưa thể xác minh phiên mua hàng</h1><p>Vui lòng tải lại để tiếp tục.</p></main>;
  return children;
}
