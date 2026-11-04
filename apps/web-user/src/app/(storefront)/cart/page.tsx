import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { CustomerPurchaseGuard } from "@/components/auth/customer-purchase-guard";
import { CartFlow } from "./cart-flow";

export const metadata: Metadata = {
  title: "Giỏ hàng",
  description: "Kiểm tra quy cách, số lượng và tạm tính giỏ hàng Ô Mạ Huế."
};

export default function CartPage() {
  return (
    <>
      <SiteHeader />
      <CustomerPurchaseGuard><Suspense fallback={<CartPageLoading />}>
        <CartFlow />
      </Suspense></CustomerPurchaseGuard>
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
