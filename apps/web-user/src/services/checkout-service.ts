import type {
  CheckoutConfirmation,
  CheckoutPreparation,
  ConfirmCheckoutInput,
  PrepareCheckoutInput,
  QuoteShippingInput,
  ShippingQuote
} from "@/lib/checkout/types";
import { CheckoutApiError } from "@/lib/checkout/types";

interface CheckoutRequestOptions {
  mockScenario?: string;
  cartScenario?: string;
}

async function requestCheckout<T>(path: string, body: unknown, options: CheckoutRequestOptions = {}): Promise<T> {
  const params = new URLSearchParams();
  if (options.mockScenario) params.set("mockScenario", options.mockScenario);
  if (options.cartScenario) params.set("cartScenario", options.cartScenario);
  const suffix = params.size ? `?${params}` : "";
  const response = await fetch(`/api/checkout/${path}${suffix}`, {
    method: "POST",
    cache: "no-store",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body)
  });
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json") ? await response.json() : {};
  if (!response.ok) throw new CheckoutApiError(response.status, payload);
  return payload as T;
}

export const checkoutService = {
  prepare(input: PrepareCheckoutInput, options?: CheckoutRequestOptions): Promise<CheckoutPreparation> {
    return requestCheckout("prepare", input, options);
  },
  quoteShipping(input: QuoteShippingInput, options?: CheckoutRequestOptions): Promise<ShippingQuote> {
    return requestCheckout("shipping-quotes", input, options);
  },
  confirm(input: ConfirmCheckoutInput, options?: CheckoutRequestOptions): Promise<CheckoutConfirmation> {
    return requestCheckout("confirm", input, options);
  }
};
