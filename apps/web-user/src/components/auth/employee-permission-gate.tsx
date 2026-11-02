"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { useEmployeeSession } from "./employee-session-provider";
import type { EmployeePermission } from "@/features/employee/auth/types";

export function EmployeePermissionGate({ permission, children }: { permission: EmployeePermission; children: ReactNode }) {
  const { hasPermission } = useEmployeeSession();
  if (hasPermission(permission)) return children;

  return <main className="admin-state" role="alert">
    <span>!</span>
    <h1>Không đủ quyền truy cập</h1>
    <p>Tài khoản nhân viên hiện tại không có permission <strong>{permission}</strong>.</p>
    <Link className="admin-primary-button" href="/admin">Về bảng điều khiển</Link>
  </main>;
}
