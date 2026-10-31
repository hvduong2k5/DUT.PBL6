import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { CartFlow } from "./cart-flow";

export const metadata: Metadata = {
  title: "Giỏ hàng",
  description: "Kiểm tra quy cách, số lượng và tạm tính giỏ hàng Ô Mạ Huế."
};

export default function CartPage() {
  return (
    <>
      <SiteHeader />
      <Suspense fallback={<CartPageLoading />}>
        <CartFlow />
      </Suspense>
      <SiteFooter />
    </>
  );
}

function CartPageLoading() {
  return (
    <main className="cart-page">
      <div className="cart-loading" aria-busy="true" aria-label="Đang tải giỏ hàng">
        <div /><div /><div />
      </div>
    </main>
  );
}
