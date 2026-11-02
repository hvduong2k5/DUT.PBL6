import type { Metadata } from "next";
import { AuthProvider } from "@/components/auth/auth-provider";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: "Ô Mạ Huế", template: "%s · Ô Mạ Huế" },
  description: "Tinh hoa bánh mứt Cố Đô và trải nghiệm thành viên Tri Kỷ."
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="vi"><body><AuthProvider>{children}</AuthProvider></body></html>;
}
