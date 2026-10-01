"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import type { Cart } from "@/lib/cart/types";
import { cartService } from "@/services/cart-service";

interface CartSummaryValue {
  itemCount: number | null;
  reload: () => Promise<void>;
  updateFromCart: (cart: Cart) => void;
}

const CartSummaryContext = createContext<CartSummaryValue | null>(null);

export function CartSummaryProvider({ children }: { children: ReactNode }) {
  const session = useCustomerSession();
  const [itemCount, setItemCount] = useState<number | null>(null);

  const reload = useCallback(async () => {
    try {
      const cart = await cartService.get();
      setItemCount(cart.itemCount);
    } catch {
      setItemCount(null);
    }
  }, []);

  useEffect(() => {
    if (session.status !== "loading") void Promise.resolve().then(reload);
  }, [reload, session.status]);

  const updateFromCart = useCallback((cart: Cart) => setItemCount(cart.itemCount), []);
  const value = useMemo(() => ({ itemCount, reload, updateFromCart }), [itemCount, reload, updateFromCart]);
  return <CartSummaryContext.Provider value={value}>{children}</CartSummaryContext.Provider>;
}

export function useCartSummary() {
  const value = useContext(CartSummaryContext);
  if (!value) throw new Error("useCartSummary must be used inside CartSummaryProvider");
  return value;
}
