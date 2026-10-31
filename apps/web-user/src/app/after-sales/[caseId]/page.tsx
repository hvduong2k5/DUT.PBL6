import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { AfterSalesCaseFlow } from "./after-sales-case-flow";

export const metadata: Metadata = { title: "Chi tiết hồ sơ hậu mãi", description: "Theo dõi Return Case và phương án xử lý hậu mãi Ô Mạ." };

export default async function AfterSalesCasePage({ params }: { params: Promise<{ caseId: string }> }) {
  const { caseId } = await params;
  return <><SiteHeader /><Suspense fallback={null}><AfterSalesCaseFlow caseId={caseId} /></Suspense><SiteFooter /></>;
}
