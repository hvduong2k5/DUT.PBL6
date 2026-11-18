"use client";

import { useEffect, useState } from "react";
import { useCustomerSession } from "@/components/auth/customer-session-provider";
import { PENDING_PAYMENT_STATUS, summarizePendingPayment, type PendingPaymentSummary } from "@/lib/orders/pending-payment";
import { orderService } from "@/services/order-service";

/**
 * Number of orders the signed-in customer has not paid yet (read from the existing order list API, filtered by status).
 * Returns null while loading, for guests, or when the list cannot be read, so callers simply render nothing.
 */
export function usePendingPaymentOrders(): PendingPaymentSummary | null {
  const { status, hasCapability } = useCustomerSession();
  const allowed = status === "authenticated" && hasCapability("ORDER_HISTORY_VIEW");
  const [summary, setSummary] = useState<PendingPaymentSummary | null>(null);

  useEffect(() => {
    let active = true;
    void Promise.resolve().then(async () => {
      if (!active) return;
      if (!allowed) {
        setSummary(null);
        return;
      }
      try {
        const page = await orderService.list({ limit: 20, status: PENDING_PAYMENT_STATUS });
        if (active) setSummary(summarizePendingPayment(page));
      } catch {
        if (active) setSummary(null);
      }
    });
    return () => { active = false; };
  }, [allowed]);

  return allowed ? summary : null;
}
