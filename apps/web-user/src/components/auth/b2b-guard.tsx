"use client";

import Link from "next/link";
import { useEffect, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useCustomerSession } from "./customer-session-provider";

export function B2BGuard({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { status, capabilityStatus, hasCapability, reload } = useCustomerSession();

  useEffect(() => {
    if (status === "guest") router.replace(`/login?returnUrl=${encodeURIComponent(pathname)}`);
  }, [pathname, router, status]);

  if (status === "authenticated" && capabilityStatus === "ready" && hasCapability("B2B_QUOTE_VIEW")) return children;
  if (status === "authenticated" && capabilityStatus === "ready") return <main className="session-state"><span>♙</span><h1>Tài khoản chưa được xác minh doanh nghiệp</h1><p>Không gian báo giá sỉ chỉ dành cho đại diện doanh nghiệp đã được Ô Mạ xác minh.</p><Link className="primary-link" href="/support/request">Liên hệ để đăng ký B2B</Link></main>;
  if (capabilityStatus === "error" || status === "error") return <main className="session-state"><span>!</span><h1>Chưa thể kiểm tra quyền B2B</h1><p>Quyền truy cập đang được khóa an toàn cho đến khi xác minh thành công.</p><button className="primary-button" type="button" onClick={() => void reload()}>Thử lại</button></main>;
  return <main className="session-state" aria-busy="true"><span className="large-spinner" /><h1>{status === "guest" ? "Đang chuyển đến đăng nhập…" : "Đang xác minh tư cách doanh nghiệp…"}</h1></main>;
}
