"use client";

import { useEffect, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "./auth-provider";

export function EmployeeAccessBoundary({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { status, profile, reload } = useAuth();

  useEffect(() => {
    if (status === "guest") router.replace(`/login?returnUrl=${encodeURIComponent(pathname || "/employee")}`);
    if (status === "authenticated" && profile?.principalType === "CUSTOMER") router.replace("/");
  }, [pathname, profile?.principalType, router, status]);

  if (status === "authenticated" && profile?.principalType === "EMPLOYEE") return children;
  if (status === "error") return <main className="employee-access-state"><span>!</span><h1>Chưa thể xác minh tài khoản nhân viên</h1><p>Không gian nội bộ được khóa cho đến khi Identity Service phản hồi.</p><button type="button" onClick={() => void reload()}>Thử lại</button></main>;
  return <main className="employee-access-state" aria-busy="true"><span className="employee-spinner" /><h1>{status === "guest" ? "Đang chuyển đến đăng nhập…" : "Đang xác minh tài khoản nhân viên…"}</h1></main>;
}
