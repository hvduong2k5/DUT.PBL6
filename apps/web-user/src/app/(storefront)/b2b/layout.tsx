import type { ReactNode } from "react";
import { B2BGuard } from "@/components/auth/b2b-guard";

export default function B2BLayout({ children }: { children: ReactNode }) {
  return <B2BGuard>{children}</B2BGuard>;
}
