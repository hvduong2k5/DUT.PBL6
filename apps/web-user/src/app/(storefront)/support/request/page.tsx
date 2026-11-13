import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { SupportRequestFlow } from "./support-request-flow";

export const metadata: Metadata = { title: "Gửi yêu cầu hỗ trợ", description: "Gửi câu hỏi và nhận mã Ticket chăm sóc khách hàng Ô Mạ." };

export default function SupportRequestPage() {
  return <><SiteHeader /><Suspense fallback={null}><SupportRequestFlow /></Suspense><SiteFooter /></>;
}
