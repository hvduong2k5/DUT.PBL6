import type { CheckoutConfirmation, CheckoutLine, CheckoutPaymentMethod, CheckoutRecipient, ShippingAddress } from "./types";
import type { SavedCheckoutAddress } from "./types";
import type { CustomerCoreAddress, CustomerCoreProfile } from "@/lib/customer/customer-core";

interface Money {
  currency_code: string;
  units: number;
  nanos: number;
}

export interface CustomerCoreCheckoutRequest {
  channel: "D2C_WEB";
  shipping_address: {
    recipient_name: string;
    phone_number: string;
    street_address: string;
    ward: string;
    district: string;
    province: string;
  };
  items: Array<{ sku_code: string; quantity: number; price: Money }>;
  payment_method: "VIETQR" | "COD";
  voucher_code?: string;
}

export interface CustomerCoreCheckoutResponse {
  order_id: string;
  status: "PENDING_PAYMENT" | "PLACED";
  subtotal_amount: Money;
  discount_amount: Money;
  shipping_fee: Money;
  final_amount: Money;
  vietqr_url: string;
  payment_expires_at: string;
}

function checkoutAreaCode(value: string, fallback: string) {
  const normalized = value.normalize("NFD").replace(/[\u0300-\u036f]/gu, "").replace(/đ/giu, "d").toUpperCase().replace(/[^A-Z0-9]+/gu, "-").replace(/^-|-$/gu, "");
  if (normalized.includes("HUE")) return fallback === "province" ? "HUE" : normalized;
  if (normalized.includes("DA-NANG")) return fallback === "province" ? "DA-NANG" : normalized;
  if (fallback === "ward") return normalized.replace(/^(?:PHUONG|XA|THI-TRAN)-/u, "");
  return normalized || fallback.toUpperCase();
}

export function mapCustomerCoreCheckoutPreferences(profile?: CustomerCoreProfile, addresses: CustomerCoreAddress[] = []) {
  const defaultRecipient = profile?.full_name && profile.phone_number ? {
    fullName: profile.full_name,
    phone: profile.phone_number,
    email: profile.email ?? ""
  } satisfies CheckoutRecipient : undefined;

  const savedAddresses: SavedCheckoutAddress[] = addresses.map((address) => ({
    addressId: address.id,
    label: address.is_default ? "Địa chỉ mặc định" : "Địa chỉ nhận hàng",
    recipient: {
      fullName: address.recipient_name,
      phone: address.phone_number,
      email: profile?.email ?? ""
    },
    address: {
      provinceCode: checkoutAreaCode(address.province, "province"),
      provinceName: address.province,
      wardCode: checkoutAreaCode(address.ward, "ward"),
      wardName: address.ward,
      addressLine: address.street_address
    },
    isDefault: address.is_default
  }));

  return { defaultRecipient, savedAddresses };
}

export function buildCustomerCoreCheckoutRequest(
  lines: CheckoutLine[],
  recipient: CheckoutRecipient,
  address: ShippingAddress,
  paymentMethod: CheckoutPaymentMethod,
  voucherCode?: string
): CustomerCoreCheckoutRequest {
  return {
    channel: "D2C_WEB",
    shipping_address: {
      recipient_name: recipient.fullName,
      phone_number: recipient.phone,
      street_address: address.addressLine,
      ward: address.wardName,
      // The source Customer API still exposes this legacy field. The current
      // customer UI follows the two-level Province/City -> Ward/Commune model.
      district: "",
      province: address.provinceName
    },
    items: lines.map((line) => ({
      sku_code: line.skuId,
      quantity: line.quantity,
      price: { currency_code: "VND", units: line.unitPriceVnd, nanos: 0 }
    })),
    payment_method: paymentMethod === "BANK_TRANSFER" ? "VIETQR" : "COD",
    ...(voucherCode ? { voucher_code: voucherCode } : {})
  };
}

export function mapCustomerCoreCheckoutConfirmation(
  response: CustomerCoreCheckoutResponse,
  paymentMethod: CheckoutPaymentMethod,
  customerMode: "GUEST" | "REGISTERED" = "GUEST"
): CheckoutConfirmation {
  return {
    customerMode,
    orderId: response.order_id,
    orderNumber: response.order_id,
    status: response.status,
    paymentStatus: "UNPAID",
    paymentMethod,
    reservationExpiresAt: response.payment_expires_at,
    subtotalVnd: response.subtotal_amount.units,
    shippingFeeVnd: response.shipping_fee.units,
    discountVnd: response.discount_amount.units,
    totalVnd: response.final_amount.units,
    nextStep: paymentMethod === "BANK_TRANSFER" ? "PAYMENT_REQUIRED" : "ORDER_PLACED",
    paymentPath: paymentMethod === "BANK_TRANSFER" ? `/payment/${encodeURIComponent(response.order_id)}` : `/orders/${encodeURIComponent(response.order_id)}/confirmation`,
    message: paymentMethod === "BANK_TRANSFER" ? "Đơn hàng đã được tạo. Vui lòng hoàn tất VietQR trước thời hạn." : "Đơn hàng đã được tạo và sẽ được thanh toán khi nhận hàng.",
    vietQrUrl: response.vietqr_url,
    paymentExpiresAt: response.payment_expires_at
  };
}
