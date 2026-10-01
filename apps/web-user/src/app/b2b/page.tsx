import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { B2BRequestFlow } from "./b2b-request-flow";

export const metadata: Metadata = { title: "Báo giá quà biếu doanh nghiệp", description: "Gửi và theo dõi yêu cầu báo giá sỉ dành cho doanh nghiệp cùng Ô Mạ." };

export default function B2BPage() {
  return <><SiteHeader /><Suspense fallback={null}><B2BRequestFlow /></Suspense><SiteFooter /></>;
}
