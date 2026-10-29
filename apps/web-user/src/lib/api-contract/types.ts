export type IsoDateTime = string;

export interface PublicErrorBody {
  code: string;
  message: string;
  requestId: string;
  retryable?: boolean;
  operation_id?: string;
  details?: Record<string, unknown>;
}

export interface CatalogMoney {
  currency_code: string;
  units: number;
  nanos: number;
}

export interface CatalogCategory {
  category_id: string;
  name: string;
  slug: string;
  description: string;
  image_url: string;
  parent_id: string | null;
}

export interface CatalogCategoriesResponse {
  categories: CatalogCategory[];
}

export interface CatalogProductSummary {
  product_id: string;
  name: string;
  slug: string;
  summary: string;
  thumbnail_url: string;
  base_price: CatalogMoney;
  ocop_star: number;
  category_name: string;
  in_stock: boolean | null;
  stock_available: number | null;
}

export interface CatalogProductsResponse {
  items: CatalogProductSummary[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface CatalogSearchResponse {
  query: string;
  items: CatalogProductSummary[];
  total: number;
}

export interface CatalogVariant {
  variant_id: string;
  sku_code: string;
  name: string;
  weight_gram: number;
  price: CatalogMoney;
  original_price: CatalogMoney;
  stock_available: number | null;
}

export interface CatalogProductDetail {
  product_id: string;
  name: string;
  slug: string;
  description: string;
  images: string[];
  ocop_star: number;
  ocop_certificate_no: string;
  story: string;
  variants: CatalogVariant[];
}

export interface OrderItem {
  sku_code: string;
  quantity: number;
  item_id?: string;
  unit_price?: number;
  product_name?: string;
  weight_grams?: number;
  catalog_version?: number;
}

export interface OrderCart {
  revision: number;
  items: OrderItem[];
}

export interface AddCartItemRequest {
  sku_code: string;
  quantity: number;
  revision: number;
}

export interface UpdateCartItemRequest {
  quantity: number;
  revision: number;
}

export type OrderPaymentMethod = "VIETQR" | "COD";

export interface OrderAddress {
  recipient_name: string;
  phone: string;
  street: string;
  ward_code: string;
  province_code: string;
}

export interface CreateCheckoutQuoteRequest {
  cart_revision: number;
  payment_method: OrderPaymentMethod;
  shipping_address_id?: string;
  address?: OrderAddress;
  voucher_code?: string;
}

export interface OrderQuote {
  catalog_snapshot_id: string;
  catalog_expires_at: IsoDateTime;
  shipping_quote_reference: string;
  quote_id: string;
  cart_revision: number;
  payment_method: OrderPaymentMethod;
  items: OrderItem[];
  address: OrderAddress;
  subtotal: number;
  shipping_fee: number;
  final_amount: number;
  expires_at: IsoDateTime;
  shipping_expires_at: IsoDateTime;
  source_address_id?: string;
  source_address_version?: number;
  customer_id?: string;
}

export interface AcceptCheckoutRequest {
  quote_id: string;
  cart_revision: number;
  payment_method: OrderPaymentMethod;
}

export type CheckoutOperationStatus =
  | "ACCEPTED"
  | "PROCESSING"
  | "WAITING_RETRY"
  | "COMPENSATING"
  | "SUCCEEDED"
  | "FAILED"
  | "MANUAL_REVIEW";

export type CheckoutOperationStage = "VALIDATING" | "ACQUIRING" | "PLACING" | "COMPENSATING" | "DONE";

export interface CheckoutOperation {
  operation_id: string;
  status: CheckoutOperationStatus;
  order_id: string | null;
  accepted_at: IsoDateTime;
  stage: CheckoutOperationStage;
  updated_at: IsoDateTime;
  order_status?: string;
  error_code?: string;
  retry_at?: IsoDateTime | null;
}

export type OrderStatus =
  | "PENDING_PAYMENT"
  | "PAYMENT_FINALIZING"
  | "PAID"
  | "CONFIRMED_COD"
  | "PROCESSING"
  | "PACKED"
  | "SHIPPED"
  | "DELIVERED"
  | "DELIVERY_FAILED"
  | "COMPLETED"
  | "CANCELLED_TIMEOUT"
  | "CANCELLED_BY_USER"
  | "CANCELLED_BY_ADMIN";

export type OrderPaymentStatus =
  | "UNPAID"
  | "PENDING"
  | "CONFIRMED"
  | "RECONCILIATION_REQUIRED"
  | "PARTIALLY_REFUNDED"
  | "REFUNDED";

export type OrderStockStatus =
  | "RESERVED"
  | "COMMITTING"
  | "COMMITTED"
  | "RELEASING"
  | "RELEASED"
  | "EXPIRED"
  | "REVERSED"
  | "UNKNOWN";

export type OrderCompensationStatus = "NOT_REQUIRED" | "PENDING" | "SUCCEEDED" | "MANUAL_REVIEW";
export type OrderCancellationStatus = "NONE" | "PENDING" | "SUCCEEDED" | "REJECTED" | "MANUAL_REVIEW";
export type OrderAllowedAction = "PAY" | "CANCEL" | "REQUEST_CANCELLATION" | "VIEW_TRACKING" | "CONTACT_SUPPORT";

export interface OwnedOrder {
  order_id: string;
  operation_id: string;
  order_code: string;
  customer_id: string | null;
  warehouse_id: string;
  payment_method: string;
  status: OrderStatus;
  payment_status: OrderPaymentStatus;
  stock_status: OrderStockStatus;
  subtotal: number;
  shipping_fee: number;
  final_amount: number;
  version: number;
  ready_generation: number;
  ready_authorized: boolean;
  operational_hold: boolean;
  has_active_case: boolean;
  payment_reference: string;
  task_id: string | null;
  shipment_id: string | null;
  seal_code: string | null;
  payment_expires_at: IsoDateTime;
  updated_at: IsoDateTime;
  created_at: IsoDateTime;
  sla_due_at: IsoDateTime | null;
  completion_due_at: IsoDateTime | null;
  items_digest: string;
  compensation_status: OrderCompensationStatus;
  cancellation_status: OrderCancellationStatus;
  allowed_actions: OrderAllowedAction[];
  snapshot?: OrderQuote | null;
}

export interface OwnedOrdersResponse {
  items: OwnedOrder[];
  next_cursor: string;
}

export type OrderTrackingResponse = Record<string, unknown>;

export interface CancelOrderRequest {
  active: boolean;
  reason: string;
}

export interface OrderCancellation {
  cancellation_id: string;
  order_id: string;
  status: Exclude<OrderCancellationStatus, "NONE">;
  compensation_status: OrderCompensationStatus;
  version: number;
  review_reference?: string | null;
}

export interface PaidCancellationRequest {
  reason: string;
}

export interface OrderCancellationReview {
  review_reference: string;
  order_id: string;
  status: "REVIEW_PENDING";
}

export interface GuestPaymentGrant {
  access_token: string;
  token_type: "Bearer";
  order_id: string;
  expires_at: IsoDateTime;
  session_expires_at: IsoDateTime;
  grant_version: number;
}

export interface PaymentReceiver {
  bank_code: string;
  bank_name: string;
  account_number: string;
  account_holder: string;
  mode: "BANK_ACCOUNT" | "OFFICIAL_VA" | "CONTENT_VA";
}

export interface PaymentInstructions {
  order_id: string;
  intent_id: string;
  provider: "SEPAY";
  method: "VIETQR";
  amount: number;
  currency: "VND";
  payment_code: string;
  receiver: PaymentReceiver;
  transfer_content: string;
  qr_url: string;
  expires_at: IsoDateTime;
  server_time: IsoDateTime;
  can_pay: true;
  version: number;
}

export type PaymentDisplayState =
  | "WAITING_PAYMENT"
  | "VERIFYING"
  | "VERIFICATION_DELAYED"
  | "FINALIZING"
  | "PAID"
  | "RECONCILIATION_REQUIRED"
  | "EXPIRED"
  | "CANCELLED"
  | "PARTIALLY_REFUNDED"
  | "REFUNDED";

export type PaymentSupportReason =
  | "VERIFICATION_TIMEOUT"
  | "UNDERPAYMENT"
  | "OVERPAYMENT"
  | "MULTIPLE_PARTIAL_PAYMENTS"
  | "EXTRA_RECEIPT"
  | "LATE_PAYMENT"
  | "OTHER_REVIEW";

export interface PaymentStatus {
  order_id: string;
  intent_id: string;
  provider: "SEPAY";
  method: "VIETQR";
  order_status: Exclude<OrderStatus, "CONFIRMED_COD">;
  payment_status: OrderPaymentStatus;
  display_state: PaymentDisplayState;
  intent_status: "ACTIVE" | "SETTLED" | "EXPIRED" | "CANCELLED";
  expected_amount: number;
  allocated_amount: number;
  refunded_amount: number;
  currency: "VND";
  expires_at: IsoDateTime;
  server_time: IsoDateTime;
  has_open_reconciliation: boolean;
  can_pay: boolean;
  version: number;
  verification_started_at: IsoDateTime | null;
  verification_due_at: IsoDateTime | null;
  support_reference: string | null;
  support_reason: PaymentSupportReason | null;
  next_poll_after_seconds: number;
}

export interface Profile {
  id?: string;
  user_id?: string;
  full_name?: string;
  phone_number?: string;
  email?: string;
  date_of_birth?: IsoDateTime | null;
  gender?: string;
  avatar_url?: string;
  preferences?: Record<string, unknown>;
  status?: string;
  version?: number;
  created_at?: IsoDateTime;
  updated_at?: IsoDateTime;
}

export interface ProfileUpdate {
  full_name: string;
  phone_number: string;
  version?: number;
}

export type ProfileAddressLabel = "HOME" | "OFFICE" | "GIFT_RECIPIENT" | "OTHER" | "GIFT";

export interface AddressWrite {
  recipient_name: string;
  phone_number: string;
  street_address: string;
  ward_code: string;
  province_code: string;
  ward_name?: string;
  province_name?: string;
  latitude?: number | null;
  longitude?: number | null;
  label?: ProfileAddressLabel;
  is_default?: boolean;
}

export type AddressUpdate = Omit<AddressWrite, "is_default">;

export type ShippingAddress = Partial<AddressWrite> & {
  id?: string;
  customer_id?: string;
  version?: number;
  is_default?: boolean;
  is_deleted?: boolean;
  created_at?: IsoDateTime;
  updated_at?: IsoDateTime;
};
