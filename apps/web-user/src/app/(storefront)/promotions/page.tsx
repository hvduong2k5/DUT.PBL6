import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { PromotionsFlow } from "./promotions-flow";

export const metadata: Metadata = { title: "Ưu đãi Tri Kỷ", description: "Voucher và ưu đãi đang phát hành tại Ô Mạ Huế." };

export default function PromotionsPage() {
  return <><SiteHeader /><Suspense fallback={<main className="promotions-page"><div className="voucher-loading" aria-busy="true"><span /><span /><span /></div></main>}><PromotionsFlow /></Suspense><SiteFooter /></>;
}
