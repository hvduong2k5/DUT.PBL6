import type { Metadata } from "next";
import { AuthProvider } from "@/components/auth/auth-provider";
import { RecommendationProvider } from "@/components/recommendation/recommendation-provider";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: "Gói Huế", template: "%s · Gói Huế" },
  description: "Mè xửng, trà Huế và quà biếu dành cho người thương."
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="vi"><body><AuthProvider><RecommendationProvider>{children}</RecommendationProvider></AuthProvider></body></html>;
}
