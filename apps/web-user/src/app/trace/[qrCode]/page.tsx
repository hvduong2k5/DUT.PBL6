import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { TraceExperience } from "./trace-experience";

export const metadata: Metadata = {
  title: "Truy xuất nguồn gốc OCOP",
  description: "Kiểm chứng lô sản xuất, nguyên liệu và chứng nhận OCOP của sản phẩm Ô Mạ Huế."
};

export default async function TracePage({ params }: { params: Promise<{ qrCode: string }> }) {
  const { qrCode } = await params;
  return <><SiteHeader /><Suspense fallback={<main className="trace-page"><section className="trace-loading" aria-busy="true"><span /><span /><span /><span /></section></main>}><TraceExperience key={qrCode} qrCode={qrCode} /></Suspense><SiteFooter /></>;
}
