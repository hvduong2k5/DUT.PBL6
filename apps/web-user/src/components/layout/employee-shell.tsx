"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { useEmployeeSession } from "@/components/auth/employee-session-provider";
import styles from "./employee-shell.module.css";

export function EmployeeShell({ children }: { children: ReactNode }) {
  const { employee, logout } = useEmployeeSession();
  const initial = employee.displayName.trim().charAt(0).toLocaleUpperCase("vi") || "N";
  return <div className={styles.root}>
    <header className={styles.header}>
      <div className={styles.brand}><span className={styles.mark}>Ô</span><div><strong>Ô MẠ</strong><small>Không gian vận hành nội bộ</small></div></div>
      <div className={styles.identity}><span className={styles.avatar}>{initial}</span><div><strong>{employee.displayName}</strong><small>{employee.email}</small></div><button className={styles.logout} type="button" onClick={() => void logout()}>Đăng xuất</button></div>
    </header>
    <div className={styles.body}>
      <aside className={styles.sidebar}><small>Khu vực nhân viên</small><nav><Link href="/employee">Tổng quan</Link></nav></aside>
      <main className={styles.content}>{children}</main>
    </div>
  </div>;
}
