import type { Metadata } from "next";
import { Be_Vietnam_Pro, Playfair_Display } from "next/font/google";
import { AuthProvider } from "@/components/auth/auth-provider";
import { RecommendationProvider } from "@/components/recommendation/recommendation-provider";
import "./globals.css";
import "@/styles/heritage.css";
import "@/styles/heritage-shell.css";
import "@/styles/heritage-home.css";
import "@/styles/heritage-pages.css";

const playfair = Playfair_Display({
  subsets: ["latin", "vietnamese"],
  weight: ["500", "600", "700"],
  style: ["normal", "italic"],
  variable: "--font-playfair",
  display: "swap"
});

const beVietnam = Be_Vietnam_Pro({
  subsets: ["latin", "vietnamese"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-be-vietnam",
  display: "swap"
});

export const metadata: Metadata = {
  title: { default: "Gói Huế", template: "%s · Gói Huế" },
  description: "Tinh hoa bánh mứt Cố Đô và trải nghiệm thành viên Tri Kỷ."
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="vi" className={`${playfair.variable} ${beVietnam.variable}`}><body><AuthProvider><RecommendationProvider>{children}</RecommendationProvider></AuthProvider></body></html>;
}
