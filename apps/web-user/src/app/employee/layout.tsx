import type { ReactNode } from "react";
import { AdminSessionProvider } from "@admin/components/auth/admin-session-provider";
import { AdminShell } from "@admin/components/layout/admin-shell";
import "@admin/app/globals.css";

export default function ProtectedAdminLayout({ children }: { children: ReactNode }) {
  return <AdminSessionProvider><AdminShell>{children}</AdminShell></AdminSessionProvider>;
}
