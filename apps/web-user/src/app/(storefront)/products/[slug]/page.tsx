import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { ProductDetailFlow } from "./product-detail-flow";

export const metadata: Metadata = {
  title: "Chi tiết sản phẩm",
  description: "Xem thông tin thực phẩm và chọn đúng quy cách sản phẩm Ô Mạ Huế."
};

export default async function ProductDetailPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  return (
    <>
      <SiteHeader />
      <Suspense fallback={<main className="product-detail-page"><ProductDetailLoading /></main>}>
        <ProductDetailFlow slug={slug} />
      </Suspense>
      <SiteFooter />
    </>
  );
}

function ProductDetailLoading() {
  return (
    <div className="detail-loading" aria-busy="true" aria-label="Đang tải chi tiết sản phẩm">
      <div className="detail-loading-gallery" />
      <div className="detail-loading-copy"><span /><span /><span /><span /></div>
    </div>
  );
}
