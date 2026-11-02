import type { ReactNode } from "react";
import { CustomerSessionProvider } from "@/components/auth/customer-session-provider";
import { CartSummaryProvider } from "@/components/cart/cart-summary-provider";

export default function StorefrontLayout({ children }: { children: ReactNode }) {
  return (
    <CustomerSessionProvider>
      <CartSummaryProvider>{children}</CartSummaryProvider>
    </CustomerSessionProvider>
  );
}
