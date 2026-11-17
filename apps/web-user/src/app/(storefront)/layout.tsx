import type { ReactNode } from "react";
import { CustomerSessionProvider } from "@/components/auth/customer-session-provider";
import { CartSummaryProvider } from "@/components/cart/cart-summary-provider";
import { AiChatWidget } from "@/components/ai-chat/ai-chat-widget";

export default function StorefrontLayout({ children }: { children: ReactNode }) {
  return (
    <CustomerSessionProvider>
      <CartSummaryProvider>{children}<AiChatWidget /></CartSummaryProvider>
    </CustomerSessionProvider>
  );
}
