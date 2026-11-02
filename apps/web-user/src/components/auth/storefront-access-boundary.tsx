"use client";

import { useEffect, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "./auth-provider";

export function StorefrontAccessBoundary({ children }: { children: ReactNode }) {
  const router = useRouter();
  const { status, profile } = useAuth();
  const employee = status === "authenticated" && profile?.principalType === "EMPLOYEE";

  useEffect(() => {
    if (employee) router.replace("/employee");
  }, [employee, router]);

  if (employee) return <main className="session-state" aria-busy="true"><span className="large-spinner" /><h1>Đang mở không gian nhân viên…</h1></main>;
  // Public catalog pages remain available if Identity is temporarily unavailable.
  // Transactional pages and BFF handlers still fail closed through Customer guards.
  return children;
}
