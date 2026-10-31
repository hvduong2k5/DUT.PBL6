import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { OrderConfirmationFlow } from "./order-confirmation-flow";

export const metadata: Metadata = { title: "Xác nhận Order", description: "Xác nhận Order đã được tiếp nhận và mở hành trình theo dõi." };

export default async function OrderConfirmationPage({ params }: { params: Promise<{ orderId: string }> }) {
  const { orderId } = await params;
  return <><SiteHeader /><Suspense fallback={null}><OrderConfirmationFlow orderId={orderId} /></Suspense><SiteFooter /></>;
}
