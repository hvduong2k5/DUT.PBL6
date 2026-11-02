import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { OrderDetailFlow } from "./order-detail-flow";

export const metadata: Metadata = { title: "Chi tiết Order", description: "Xem snapshot và hành trình Order Ô Mạ Huế." };

export default async function OrderDetailPage({ params }: { params: Promise<{ orderId: string }> }) {
  const { orderId } = await params;
  return <><SiteHeader /><Suspense fallback={null}><OrderDetailFlow orderId={orderId} /></Suspense><SiteFooter /></>;
}
