import type { ReactNode } from "react";
import { CustomerSessionProvider } from "@/components/auth/customer-session-provider";
import { StorefrontAccessBoundary } from "@/components/auth/storefront-access-boundary";
import { CartSummaryProvider } from "@/components/cart/cart-summary-provider";

export default function StorefrontLayout({ children }: { children: ReactNode }) {
  return (
    <StorefrontAccessBoundary>
      <CustomerSessionProvider>
        <CartSummaryProvider>{children}</CartSummaryProvider>
      </CustomerSessionProvider>
    </StorefrontAccessBoundary>
  );
}
