import type { Metadata } from "next";
import { AdminSessionProvider } from "@/components/auth/admin-session-provider";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: "Ô Mạ Admin", template: "%s · Ô Mạ Admin" },
  description: "Không gian vận hành nội bộ Ô Mạ."
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="vi"><body><AdminSessionProvider>{children}</AdminSessionProvider></body></html>;
}
