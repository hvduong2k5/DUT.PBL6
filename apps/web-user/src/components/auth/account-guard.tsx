"use client";

import Link from "next/link";
import { useEffect, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import type { CustomerCapability } from "@/lib/auth/capabilities";
import { useCustomerSession } from "./customer-session-provider";

function capabilityFor(pathname: string): CustomerCapability {
  if (pathname.startsWith("/account/addresses")) return "ADDRESS_MANAGE";
  if (pathname.startsWith("/account/orders")) return "ORDER_HISTORY_VIEW";
  if (pathname.startsWith("/account/loyalty")) return "LOYALTY_VIEW";
  return "PROFILE_MANAGE";
}

export function AccountGuard({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { status, capabilityStatus, hasCapability, reload } = useCustomerSession();

  useEffect(() => {
    if (status === "guest") router.replace(`/login?returnUrl=${encodeURIComponent(pathname)}`);
  }, [pathname, router, status]);

  if (status === "authenticated" && capabilityStatus === "ready" && hasCapability(capabilityFor(pathname))) return children;
  if (status === "authenticated" && capabilityStatus === "ready") return <main className="session-state"><span>!</span><h1>Tài khoản chưa có quyền truy cập</h1><p>Chức năng này không thuộc các capability hiện tại của tài khoản.</p><Link className="primary-link" href="/">Về trang chủ</Link></main>;
  if (status === "authenticated" && capabilityStatus === "error") return <main className="session-state"><span>!</span><h1>Chưa thể kiểm tra quyền Customer</h1><p>Không thể tải capability projection lúc này. Các chức năng cần quyền đang được khóa an toàn.</p><button className="primary-button" type="button" onClick={() => void reload()}>Thử lại</button></main>;
  if (status === "error") return <main className="session-state"><span>!</span><h1>Chưa thể kiểm tra phiên đăng nhập</h1><p>Không thể kết nối dịch vụ xác thực lúc này.</p><button className="primary-button" type="button" onClick={() => void reload()}>Thử lại</button></main>;
  return <main className="session-state" aria-busy="true"><span className="large-spinner" /><h1>{status === "guest" ? "Đang chuyển đến đăng nhập…" : "Đang kiểm tra phiên đăng nhập…"}</h1></main>;
}
