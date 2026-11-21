"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import type { Cart } from "@/lib/cart/types";
import { cartService } from "@/services/cart-service";
import { MVP_FEATURE_GATES } from "@/lib/mvp-feature-gates";
import { requestPublicJson } from "@/lib/api-contract";
import type { CartMergeResult } from "@/lib/cart/merge";

interface CartSummaryValue {
  mergeResult: CartMergeResult | null;
  mergeError: string;
  mergePending: boolean;
  retryMerge: () => Promise<void>;
  itemCount: number | null;
  reload: () => Promise<void>;
  updateFromCart: (cart: Cart) => void;
}

const CartSummaryContext = createContext<CartSummaryValue | null>(null);

export function CartSummaryProvider({ children }: { children: ReactNode }) {
  const session = useCustomerSession();
  const [itemCount, setItemCount] = useState<number | null>(null);
  const [mergeResult, setMergeResult] = useState<CartMergeResult | null>(null);
  const [mergeError, setMergeError] = useState("");
  const [mergePending, setMergePending] = useState(false);
  const mergeAttemptedFor = useRef<string | null>(null);
  const mergeActor = useRef<string | undefined>(undefined);

  useEffect(() => {
    mergeActor.current=session.status==="authenticated" ? session.principal?.principalId : undefined;
    if (session.status==="guest" || session.status==="forbidden") void Promise.resolve().then(()=>{setMergeResult(null);setMergeError("");setMergePending(false);});
  },[session.status,session.principal?.principalId]);

  const reload = useCallback(async () => {
    try {
      const cart = await cartService.get();
      setItemCount(cart.itemCount);
    } catch {
      setItemCount(null);
    }
  }, []);

  const retryMerge = useCallback(async () => {
    const actor=mergeActor.current;if (!actor) return;
    setMergePending(true);setMergeError("");
    try {
      const response=await requestPublicJson<CartMergeResult>("/api/cart/merge",{method:"POST",credentials:"include",body:{}});
      if (response.status===200 && mergeActor.current===actor) {
        setMergeResult(response.data);
        await reload();
        window.dispatchEvent(new Event("guest-cart-merged"));
      }
    } catch {if(mergeActor.current===actor)setMergeError("Bạn đã đăng nhập. Giỏ hàng khách chưa được gộp; hãy thử lại.");}
    finally {if(mergeActor.current===actor)setMergePending(false);}
  },[reload]);

  useEffect(() => {
    if (session.status!=="authenticated") {mergeAttemptedFor.current=null;return;}
    const id=session.principal?.principalId;
    if (session.customerAccess && MVP_FEATURE_GATES.guestOrder && id && mergeAttemptedFor.current!==id) {
      mergeAttemptedFor.current=id;
      void Promise.resolve().then(retryMerge);
    }
  },[retryMerge,session.status,session.customerAccess,session.principal?.principalId]);

  useEffect(() => {
    if (session.status === "authenticated" && session.customerAccess || session.status === "guest" && MVP_FEATURE_GATES.guestOrder) void Promise.resolve().then(reload);
    else if (session.status !== "loading") void Promise.resolve().then(() => setItemCount(null));
  }, [reload, session.customerAccess, session.status]);

  const updateFromCart = useCallback((cart: Cart) => setItemCount(cart.itemCount), []);
  const value = useMemo(() => ({ itemCount, reload, updateFromCart,mergeResult,mergeError,mergePending,retryMerge }), [itemCount, reload, updateFromCart,mergeResult,mergeError,mergePending,retryMerge]);
  return <CartSummaryContext.Provider value={value}>{children}</CartSummaryContext.Provider>;
}

export function useCartSummary() {
  const value = useContext(CartSummaryContext);
  if (!value) throw new Error("useCartSummary must be used inside CartSummaryProvider");
  return value;
}
