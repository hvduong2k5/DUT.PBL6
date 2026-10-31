import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { B2BQuoteFlow } from "./quote-flow";

export const metadata: Metadata = { title: "Chi tiết báo giá B2B", description: "Xem điều khoản và chấp thuận báo giá doanh nghiệp Ô Mạ." };

export default async function B2BQuotePage({ params }: { params: Promise<{ quoteId: string }> }) {
  return <><SiteHeader /><Suspense fallback={null}><B2BQuoteFlow quoteId={(await params).quoteId} /></Suspense><SiteFooter /></>;
}
