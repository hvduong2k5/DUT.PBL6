import "@/styles/storefront-heritage.css";
import "@/styles/customer-renovation.css";
import type { ReactNode } from "react";
import { CustomerSessionProvider } from "@/components/auth/customer-session-provider";
import { CartSummaryProvider } from "@/components/cart/cart-summary-provider";

export default function StorefrontLayout({
  children,
}: {
  children: ReactNode;
}) {
  return (
    <CustomerSessionProvider>
      <CartSummaryProvider>
        <div className="customer-storefront">{children}</div>
      </CartSummaryProvider>
    </CustomerSessionProvider>
  );
}
