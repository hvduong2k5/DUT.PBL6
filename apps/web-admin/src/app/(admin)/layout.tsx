import type { ReactNode } from "react";
import { AdminShell } from "@/components/layout/admin-shell";

export default function ProtectedAdminLayout({ children }: { children: ReactNode }) {
  return <AdminShell>{children}</AdminShell>;
}
