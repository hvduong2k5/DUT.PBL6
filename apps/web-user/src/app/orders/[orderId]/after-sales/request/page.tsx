import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { AfterSalesRequestFlow } from "./after-sales-request-flow";

export const metadata: Metadata = { title: "Tạo yêu cầu hậu mãi", description: "Tạo hồ sơ đổi, trả hoặc hoàn tiền cho Order Ô Mạ." };

export default async function AfterSalesRequestPage({ params }: { params: Promise<{ orderId: string }> }) {
  const { orderId } = await params;
  return <><SiteHeader /><Suspense fallback={null}><AfterSalesRequestFlow orderId={orderId} /></Suspense><SiteFooter /></>;
}
