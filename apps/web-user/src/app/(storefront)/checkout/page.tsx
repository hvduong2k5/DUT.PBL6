import type { Metadata } from "next";
import { Suspense } from "react";
import { SiteFooter } from "@/components/layout/site-footer";
import { SiteHeader } from "@/components/layout/site-header";
import { CustomerPurchaseGuard } from "@/components/auth/customer-purchase-guard";
import { CheckoutFlow } from "./checkout-flow";

export const metadata: Metadata = {
  title: "Thông tin và thanh toán",
  description: "Kiểm tra thông tin nhận hàng, phí vận chuyển và xác nhận đơn Ô Mạ Huế."
};

export default function CheckoutPage() {
  return (
    <>
      <SiteHeader />
      <CustomerPurchaseGuard><Suspense fallback={<CheckoutPageLoading />}>
        <CheckoutFlow />
      </Suspense></CustomerPurchaseGuard>
      <SiteFooter />
    </>
  );
}

function CheckoutPageLoading() {
  return <main className="checkout-page"><div className="checkout-loading" aria-busy="true" aria-label="Đang chuẩn bị Checkout"><div /><div /></div></main>;
}
