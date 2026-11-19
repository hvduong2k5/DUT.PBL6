import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { CatalogFlow } from "./catalog-flow";

export const metadata: Metadata = {
  title: "Khám phá sản phẩm",
  description: "Khám phá bánh mứt, trà và quà tặng mang phong vị Cố Đô Huế."
};

export default function ProductsPage() {
  return (
    <>
      <SiteHeader />
      <Suspense fallback={<main className="catalog-page"><div className="catalog-loading"><span className="large-spinner" /><p>Đang mở kho thức quà Cố Đô…</p></div></main>}>
        <CatalogFlow />
      </Suspense>
      <SiteFooter />
    </>
  );
}
