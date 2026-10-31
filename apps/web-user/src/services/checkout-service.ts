import { requestPublicJson } from "@/lib/api-contract";
import type { CheckoutOperation, OrderQuote } from "@/lib/api-contract";
import { cartService } from "@/services/cart-service";
import { customerService } from "@/services/customer-service";
import {
  buildMvpAcceptRequest,
  buildMvpQuoteRequest,
  mapMvpCheckoutOperation,
  mapMvpCheckoutPreparation,
  mapMvpCheckoutQuote
} from "@/lib/checkout/mvp";
import type {
  CheckoutRecipient,
  MvpCheckoutOperationView,
  MvpCheckoutPaymentMethod,
  MvpCheckoutPreparation,
  MvpCheckoutQuote,
  ShippingAddress
} from "@/lib/checkout/types";

interface CheckoutRequestOptions {
  mockScenario?: string;
  cartScenario?: string;
}

function endpoint(path: string, mockScenario?: string): string {
  const params = new URLSearchParams();
  if (mockScenario) params.set("mockScenario", mockScenario);
  return `/api/checkout${path}${params.size ? `?${params}` : ""}`;
}

function wait(milliseconds: number): Promise<void> {
  return new Promise((resolve) => globalThis.setTimeout(resolve, milliseconds));
}

async function getCheckoutOperation(operationId: string, options: CheckoutRequestOptions = {}) {
  const response = await requestPublicJson<CheckoutOperation>(endpoint(`/operations/${encodeURIComponent(operationId)}`, options.mockScenario), {
    credentials: "include"
  });
  return {
    operation: mapMvpCheckoutOperation(response.data),
    retryAfterSeconds: response.retryAfterSeconds ?? 2
  };
}

export const checkoutService = {
  async prepare(options: CheckoutRequestOptions = {}): Promise<MvpCheckoutPreparation> {
    const [cart, locations, profileResult, addressesResult] = await Promise.all([
      cartService.get(options.cartScenario),
      customerService.getLocations(),
      customerService.getProfile().then((value) => ({ ok: true as const, value })).catch(() => ({ ok: false as const })),
      customerService.getAddresses().then((value) => ({ ok: true as const, value })).catch(() => ({ ok: false as const }))
    ]);
    const profile = profileResult.ok ? profileResult.value : undefined;
    const addresses = addressesResult.ok ? addressesResult.value.items : [];
    const warning = profileResult.ok && addressesResult.ok
      ? undefined
      : "Chưa thể tải đầy đủ hồ sơ hoặc sổ địa chỉ. Bạn vẫn có thể nhập thông tin nhận hàng cho đơn này.";
    return mapMvpCheckoutPreparation(cart, locations.provinces, profile, addresses, warning);
  },

  async createQuote(input: {
    cartRevision: number;
    paymentMethod: MvpCheckoutPaymentMethod;
    selectedAddressId?: string;
    recipient: CheckoutRecipient;
    address: ShippingAddress;
    voucherCode?: string;
  }, options: CheckoutRequestOptions = {}): Promise<MvpCheckoutQuote> {
    const response = await requestPublicJson<OrderQuote>(endpoint("/quote", options.mockScenario), {
      method: "POST",
      credentials: "include",
      body: buildMvpQuoteRequest(input)
    });
    return mapMvpCheckoutQuote(response.data);
  },

  async accept(quote: MvpCheckoutQuote, idempotencyKey: string, options: CheckoutRequestOptions = {}) {
    const response = await requestPublicJson<CheckoutOperation>(endpoint("", options.mockScenario), {
      method: "POST",
      credentials: "include",
      idempotencyKey,
      body: buildMvpAcceptRequest(quote)
    });
    return {
      operation: mapMvpCheckoutOperation(response.data),
      retryAfterSeconds: response.retryAfterSeconds ?? 2
    };
  },

  getOperation: getCheckoutOperation,

  async waitForOperation(
    initial: MvpCheckoutOperationView,
    initialRetryAfterSeconds: number,
    options: CheckoutRequestOptions = {},
    onUpdate?: (operation: MvpCheckoutOperationView) => void
  ): Promise<MvpCheckoutOperationView> {
    let current = initial;
    let retryAfterSeconds = initialRetryAfterSeconds;
    for (let attempt = 0; attempt < 30 && !current.terminal; attempt += 1) {
      await wait(Math.min(5, Math.max(1, retryAfterSeconds)) * 1000);
      const next = await getCheckoutOperation(current.operationId, options);
      current = next.operation;
      retryAfterSeconds = next.retryAfterSeconds;
      onUpdate?.(current);
    }
    return current;
  }
};
